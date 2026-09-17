// Package company manages companies, the organisations contacts belong to, and the
// optional parent/child hierarchy between them.
package company

import (
	"database/sql"
	"embed"
	"errors"
	"strings"

	"github.com/abhinavxd/libredesk/internal/company/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// maxListPageSize caps how many companies a single list request can return.
const maxListPageSize = 100

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles company related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetCompanies            *sqlx.Stmt `query:"get-companies"`
	GetCompaniesCompact     *sqlx.Stmt `query:"get-companies-compact"`
	GetCompaniesCompactByID *sqlx.Stmt `query:"get-companies-compact-by-ids"`
	GetCompany              *sqlx.Stmt `query:"get-company"`
	InsertCompany           *sqlx.Stmt `query:"insert-company"`
	UpdateCompany           *sqlx.Stmt `query:"update-company"`
	DeleteCompany           *sqlx.Stmt `query:"delete-company"`
	IsCompanyDescendant     *sqlx.Stmt `query:"is-company-descendant"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves companies matching query, optionally limited to the children of parentID.
func (m *Manager) GetAll(query string, parentID, page, pageSize int) ([]models.Company, error) {
	page, pageSize = clampPagination(page, pageSize)
	var companies = make([]models.Company, 0)
	if err := m.q.GetCompanies.Select(&companies, query, parentID, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// GetAllCompact retrieves companies with just id and name, all of them when pageSize is 0.
func (m *Manager) GetAllCompact(query string, page, pageSize int) ([]models.CompanyCompact, error) {
	var companies = make([]models.CompanyCompact, 0)
	if err := m.q.GetCompaniesCompact.Select(&companies, query, pageSize, dbutil.PageOffset(page, pageSize)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// GetAllCompactByIDs retrieves the companies with the given IDs with just id and name.
func (m *Manager) GetAllCompactByIDs(ids []int) ([]models.CompanyCompact, error) {
	var companies = make([]models.CompanyCompact, 0)
	if err := m.q.GetCompaniesCompactByID.Select(&companies, pq.Array(ids)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return companies, nil
		}
		m.lo.Error("error fetching companies by ids", "error", err)
		return companies, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return companies, nil
}

// Get retrieves a company by ID.
func (m *Manager) Get(id int) (models.Company, error) {
	var company models.Company
	if err := m.q.GetCompany.Get(&company, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return company, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundCompany"), nil)
		}
		m.lo.Error("error fetching company", "id", id, "error", err)
		return company, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return company, nil
}

// Create creates a company, optionally under a parent.
func (m *Manager) Create(company models.Company) (models.Company, error) {
	name, err := m.validName(company.Name)
	if err != nil {
		return models.Company{}, err
	}

	var id int
	if err := m.q.InsertCompany.QueryRow(name, company.Description, company.Website, company.PhoneNumberCountryCode,
		company.PhoneNumber, company.Country, company.ParentID).Scan(&id); err != nil {
		return models.Company{}, m.writeError(err, "error inserting company")
	}
	return m.Get(id)
}

// Update updates a company, rejecting a parent that would put the company inside its own subtree.
func (m *Manager) Update(id int, company models.Company) (models.Company, error) {
	name, err := m.validName(company.Name)
	if err != nil {
		return models.Company{}, err
	}
	if err := m.validParent(id, company.ParentID); err != nil {
		return models.Company{}, err
	}

	res, err := m.q.UpdateCompany.Exec(id, name, company.Description, company.Website, company.PhoneNumberCountryCode,
		company.PhoneNumber, company.Country, company.ParentID)
	if err != nil {
		return models.Company{}, m.writeError(err, "error updating company")
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return models.Company{}, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundCompany"), nil)
	}
	return m.Get(id)
}

// Delete deletes a company. Its contacts and child companies survive: the contacts become
// unaffiliated and the children become roots, per ON DELETE SET NULL on both foreign keys.
func (m *Manager) Delete(id int) error {
	res, err := m.q.DeleteCompany.Exec(id)
	if err != nil {
		m.lo.Error("error deleting company", "id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundCompany"), nil)
	}
	return nil
}

// validName trims the name and rejects an empty one.
func (m *Manager) validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", "name"), nil)
	}
	return name, nil
}

// validParent rejects a company being its own parent, or being moved under one of its own
// descendants - either would cut the resulting cycle loose from every root in the tree.
func (m *Manager) validParent(id int, parentID null.Int) error {
	if !parentID.Valid {
		return nil
	}
	if parentID.Int == id {
		return envelope.NewError(envelope.InputError, m.i18n.T("company.parentCannotBeSelf"), nil)
	}

	var isDescendant bool
	if err := m.q.IsCompanyDescendant.Get(&isDescendant, id, parentID.Int); err != nil {
		m.lo.Error("error checking company descendants", "id", id, "parent_id", parentID.Int, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if isDescendant {
		return envelope.NewError(envelope.InputError, m.i18n.T("company.parentCannotBeDescendant"), nil)
	}
	return nil
}

// writeError maps an insert or update failure to the constraint the caller actually violated.
func (m *Manager) writeError(err error, logMsg string) error {
	if dbutil.IsUniqueViolationError(err) {
		return envelope.NewError(envelope.InputError, m.i18n.T("company.alreadyExistsWithName"), nil)
	}
	if dbutil.IsForeignKeyError(err) {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCompany"), nil)
	}
	m.lo.Error(logMsg, "error", err)
	return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
}

// clampPagination keeps a list request inside sane bounds.
func clampPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}
	return page, pageSize
}
