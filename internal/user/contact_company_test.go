package user

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
)

func insertCompany(t *testing.T, db *sqlx.DB, name string) int {
	t.Helper()
	var id int
	if err := db.QueryRow(`INSERT INTO companies (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		t.Fatalf("inserting company: %v", err)
	}
	return id
}

func companyOf(t *testing.T, db *sqlx.DB, contactID int) null.Int {
	t.Helper()
	var companyID null.Int
	if err := db.Get(&companyID, `SELECT company_id FROM users WHERE id = $1`, contactID); err != nil {
		t.Fatalf("reading contact company: %v", err)
	}
	return companyID
}

// CreateContact and UpdateContact are the agent-facing writes, and the only ones that
// carry a company. Channel-created contacts go through ResolveContact and have none.
func TestCreateAndUpdateContactCompany(t *testing.T) {
	u, db := newTestManager(t)
	acme := insertCompany(t, db, "Acme Industries")
	northwind := insertCompany(t, db, "Northwind MSP")

	contact := &models.User{
		Email:     null.StringFrom("worker@acme.example"),
		FirstName: "Worker",
		LastName:  "One",
		CompanyID: null.IntFrom(acme),
	}
	if err := u.CreateContact(contact); err != nil {
		t.Fatalf("creating contact: %v", err)
	}
	if got := companyOf(t, db, contact.ID); !got.Valid || got.Int != acme {
		t.Fatalf("expected company %d on create, got %+v", acme, got)
	}

	// Moved to another company.
	moved := models.User{
		FirstName: "Worker",
		LastName:  "One",
		Email:     null.StringFrom("worker@acme.example"),
		CompanyID: null.IntFrom(northwind),
	}
	if err := u.UpdateContact(contact.ID, moved); err != nil {
		t.Fatalf("updating contact: %v", err)
	}
	if got := companyOf(t, db, contact.ID); !got.Valid || got.Int != northwind {
		t.Fatalf("expected company %d after update, got %+v", northwind, got)
	}

	// Cleared back to no company at all.
	cleared := moved
	cleared.CompanyID = null.Int{}
	if err := u.UpdateContact(contact.ID, cleared); err != nil {
		t.Fatalf("clearing contact company: %v", err)
	}
	if got := companyOf(t, db, contact.ID); got.Valid {
		t.Fatalf("expected the company to be cleared, got %+v", got)
	}
}

// A contact with no company is the freelancer case and has to keep working.
func TestCreateContactWithoutCompany(t *testing.T) {
	u, db := newTestManager(t)

	contact := &models.User{
		Email:     null.StringFrom("solo@example.com"),
		FirstName: "Solo",
	}
	if err := u.CreateContact(contact); err != nil {
		t.Fatalf("creating contact: %v", err)
	}
	if got := companyOf(t, db, contact.ID); got.Valid {
		t.Fatalf("expected no company, got %+v", got)
	}
}

// Deleting the company leaves the contact and its conversations alone.
func TestDeletingCompanyUnaffiliatesContact(t *testing.T) {
	u, db := newTestManager(t)
	acme := insertCompany(t, db, "Acme Industries")

	contact := &models.User{
		Email:     null.StringFrom("worker@acme.example"),
		FirstName: "Worker",
		CompanyID: null.IntFrom(acme),
	}
	if err := u.CreateContact(contact); err != nil {
		t.Fatalf("creating contact: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM companies WHERE id = $1`, acme); err != nil {
		t.Fatalf("deleting company: %v", err)
	}

	if got := companyOf(t, db, contact.ID); got.Valid {
		t.Fatalf("expected the contact to be unaffiliated, got %+v", got)
	}
	if _, err := u.GetContactOrVisitor(contact.ID, ""); err != nil {
		t.Fatalf("expected the contact to survive its company: %v", err)
	}
}
