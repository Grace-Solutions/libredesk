package company

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/company/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "company")
	lo := logf.New(logf.Opts{})
	mgr, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatalf("creating company manager: %v", err)
	}
	return mgr, db
}

func create(t *testing.T, m *Manager, name string, parentID null.Int) models.Company {
	t.Helper()
	company, err := m.Create(models.Company{Name: name, ParentID: parentID})
	if err != nil {
		t.Fatalf("creating company %q: %v", name, err)
	}
	return company
}

// assertInputError fails unless err is an input error, i.e. the caller's fault rather than ours.
func assertInputError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	envErr, ok := err.(envelope.Error)
	if !ok {
		t.Fatalf("expected an envelope error, got %T: %v", err, err)
	}
	if envErr.ErrorType != envelope.InputError {
		t.Fatalf("expected an input error, got %v: %v", envErr.ErrorType, err)
	}
}

func TestCreateTrimsNameAndRejectsEmpty(t *testing.T) {
	m, _ := newTestManager(t)

	company := create(t, m, "  Acme Industries  ", null.Int{})
	if company.Name != "Acme Industries" {
		t.Fatalf("expected the name to be trimmed, got %q", company.Name)
	}

	_, err := m.Create(models.Company{Name: "   "})
	assertInputError(t, err)
}

func TestCreateRejectsDuplicateNameRegardlessOfCase(t *testing.T) {
	m, _ := newTestManager(t)
	create(t, m, "Acme Industries", null.Int{})

	_, err := m.Create(models.Company{Name: "ACME industries"})
	assertInputError(t, err)
}

func TestCreateRejectsUnknownParent(t *testing.T) {
	m, _ := newTestManager(t)

	_, err := m.Create(models.Company{Name: "Orphan", ParentID: null.IntFrom(4242)})
	assertInputError(t, err)
}

func TestUpdateRejectsSelfAsParent(t *testing.T) {
	m, _ := newTestManager(t)
	company := create(t, m, "Acme Industries", null.Int{})

	_, err := m.Update(company.ID, models.Company{Name: company.Name, ParentID: null.IntFrom(company.ID)})
	assertInputError(t, err)
}

// An MSP moved under one of the companies it serves would take the whole chain with it,
// leaving a cycle that no tree walk from a root can reach.
func TestUpdateRejectsParentInsideOwnSubtree(t *testing.T) {
	m, _ := newTestManager(t)
	msp := create(t, m, "Northwind MSP", null.Int{})
	client := create(t, m, "Client Co", null.IntFrom(msp.ID))
	subsidiary := create(t, m, "Client Subsidiary", null.IntFrom(client.ID))

	// One level down.
	_, err := m.Update(msp.ID, models.Company{Name: msp.Name, ParentID: null.IntFrom(client.ID)})
	assertInputError(t, err)

	// And two, which a self-check alone would let through.
	_, err = m.Update(msp.ID, models.Company{Name: msp.Name, ParentID: null.IntFrom(subsidiary.ID)})
	assertInputError(t, err)

	// Moving in the other direction is legitimate and must still work.
	if _, err := m.Update(subsidiary.ID, models.Company{Name: subsidiary.Name, ParentID: null.IntFrom(msp.ID)}); err != nil {
		t.Fatalf("re-parenting a child upwards: %v", err)
	}
}

func TestGetReportsParentAndCounts(t *testing.T) {
	m, db := newTestManager(t)
	msp := create(t, m, "Northwind MSP", null.Int{})
	client := create(t, m, "Client Co", null.IntFrom(msp.ID))
	insertContact(t, db, "a@client.example", client.ID)
	insertContact(t, db, "b@client.example", client.ID)

	got, err := m.Get(client.ID)
	if err != nil {
		t.Fatalf("getting company: %v", err)
	}
	if got.ParentName.String != msp.Name {
		t.Fatalf("expected parent name %q, got %q", msp.Name, got.ParentName.String)
	}
	if got.ContactCount != 2 {
		t.Fatalf("expected 2 contacts, got %d", got.ContactCount)
	}

	parent, err := m.Get(msp.ID)
	if err != nil {
		t.Fatalf("getting parent company: %v", err)
	}
	if parent.ChildCount != 1 {
		t.Fatalf("expected 1 child company, got %d", parent.ChildCount)
	}
	if parent.ParentName.Valid {
		t.Fatalf("expected a root company to have no parent name, got %q", parent.ParentName.String)
	}
}

func TestGetAllListsChildrenOfOneParent(t *testing.T) {
	m, _ := newTestManager(t)
	msp := create(t, m, "Northwind MSP", null.Int{})
	create(t, m, "Client One", null.IntFrom(msp.ID))
	create(t, m, "Client Two", null.IntFrom(msp.ID))
	create(t, m, "Unrelated Co", null.Int{})

	children, err := m.GetAll("", msp.ID, 1, 30)
	if err != nil {
		t.Fatalf("listing children: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	all, err := m.GetAll("", 0, 1, 30)
	if err != nil {
		t.Fatalf("listing all companies: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 companies, got %d", len(all))
	}
	if all[0].Total != 4 {
		t.Fatalf("expected a total of 4, got %d", all[0].Total)
	}
}

// Deleting a company must not take its contacts or its sub-companies with it.
func TestDeleteUnaffiliatesContactsAndPromotesChildren(t *testing.T) {
	m, db := newTestManager(t)
	msp := create(t, m, "Northwind MSP", null.Int{})
	client := create(t, m, "Client Co", null.IntFrom(msp.ID))
	contactID := insertContact(t, db, "a@client.example", msp.ID)

	if err := m.Delete(msp.ID); err != nil {
		t.Fatalf("deleting company: %v", err)
	}

	promoted, err := m.Get(client.ID)
	if err != nil {
		t.Fatalf("getting the promoted child: %v", err)
	}
	if promoted.ParentID.Valid {
		t.Fatalf("expected the child to become a root, got parent %d", promoted.ParentID.Int)
	}

	var companyID null.Int
	if err := db.Get(&companyID, `SELECT company_id FROM users WHERE id = $1`, contactID); err != nil {
		t.Fatalf("reading the contact: %v", err)
	}
	if companyID.Valid {
		t.Fatalf("expected the contact to be unaffiliated, got company %d", companyID.Int)
	}
}

func TestDeleteUnknownCompanyIsNotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.Delete(4242)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	envErr, ok := err.(envelope.Error)
	if !ok || envErr.ErrorType != envelope.NotFoundError {
		t.Fatalf("expected a not-found error, got %v", err)
	}
}

func insertContact(t *testing.T, db *sqlx.DB, email string, companyID int) int {
	t.Helper()
	var id int
	err := db.QueryRow(`
		INSERT INTO users (email, type, first_name, "password", company_id)
		VALUES ($1, 'contact', 'Test', 'x', $2) RETURNING id`, email, companyID).Scan(&id)
	if err != nil {
		t.Fatalf("inserting contact: %v", err)
	}
	return id
}
