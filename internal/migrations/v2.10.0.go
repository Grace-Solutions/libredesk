package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// companyPermissionsV2_10_0 are granted to every role that can already manage contacts,
// so existing admins keep working without editing roles by hand.
var companyPermissionsV2_10_0 = []string{
	"companies:read",
	"companies:write",
	"companies:delete",
}

func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS companies (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			parent_id BIGINT NULL REFERENCES companies(id) ON DELETE SET NULL ON UPDATE CASCADE,
			"name" TEXT NOT NULL,
			description TEXT NULL,
			website TEXT NULL,
			phone_number_country_code TEXT NULL,
			phone_number TEXT NULL,
			country TEXT NULL,
			CONSTRAINT constraint_companies_on_name CHECK (length("name") > 0 AND length("name") <= 140),
			CONSTRAINT constraint_companies_on_description CHECK (length(description) <= 300),
			CONSTRAINT constraint_companies_on_website CHECK (length(website) <= 300),
			CONSTRAINT constraint_companies_on_phone_number CHECK (length(phone_number) <= 20),
			CONSTRAINT constraint_companies_on_phone_number_country_code CHECK (length(phone_number_country_code) <= 10),
			CONSTRAINT constraint_companies_on_country CHECK (length(country) <= 140),
			CONSTRAINT constraint_companies_on_parent_id_not_self CHECK (parent_id IS NULL OR parent_id <> id)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_companies_on_name ON companies (lower("name"));
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_companies_on_parent_id ON companies(parent_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS company_id BIGINT NULL
			REFERENCES companies(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_users_on_company_id ON users(company_id) WHERE company_id IS NOT NULL;
	`); err != nil {
		return err
	}

	for _, permission := range companyPermissionsV2_10_0 {
		if _, err := db.Exec(`
			UPDATE roles
			SET permissions = array_append(permissions, $1)
			WHERE 'contacts:write' = ANY(permissions)
			AND NOT ($1 = ANY(permissions));
		`, permission); err != nil {
			return err
		}
	}
	return nil
}
