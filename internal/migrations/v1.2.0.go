package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_2_0 adds organizations, organization_members, and conversation organization_id.
func V1_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			"name" TEXT NOT NULL,
			description TEXT NULL,
			CONSTRAINT constraint_organizations_on_name CHECK (length("name") <= 140),
			CONSTRAINT constraint_organizations_on_description CHECK (length(description) <= 300)
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS organization_members (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			organization_id INT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			share_tickets_by_default BOOLEAN DEFAULT false NOT NULL,
			CONSTRAINT constraint_organization_members_on_organization_id_and_contact_id_unique UNIQUE (organization_id, contact_id)
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_organization_members_on_contact_id ON organization_members(contact_id);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE conversations
		ADD COLUMN IF NOT EXISTS organization_id INT REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_conversations_on_organization_id ON conversations(organization_id);`)
	if err != nil {
		return err
	}

	// Add organizations:manage to Admin role if not present.
	_, err = db.Exec(`
		UPDATE roles SET permissions = array_append(permissions, 'organizations:manage')
		WHERE name = 'Admin' AND NOT ('organizations:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	_ = fs
	_ = ko
	return nil
}
