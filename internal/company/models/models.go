package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

const CompanyModel = "company"

// Company is an organisation contacts can belong to. A company may itself sit under
// a parent company, e.g. an MSP holding one contract that covers several companies.
type Company struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	Name                   string      `db:"name" json:"name"`
	Description            null.String `db:"description" json:"description"`
	Website                null.String `db:"website" json:"website"`
	PhoneNumberCountryCode null.String `db:"phone_number_country_code" json:"phone_number_country_code"`
	PhoneNumber            null.String `db:"phone_number" json:"phone_number"`
	Country                null.String `db:"country" json:"country"`

	ParentID   null.Int    `db:"parent_id" json:"parent_id"`
	ParentName null.String `db:"parent_name" json:"parent_name"`

	// Counts of what hangs off this company, for the list and detail views.
	ContactCount int `db:"contact_count" json:"contact_count"`
	ChildCount   int `db:"child_count" json:"child_count"`

	Total int `db:"total" json:"-"`
}

// CompanyCompact is the minimal shape for pickers and for embedding on a contact.
type CompanyCompact struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}
