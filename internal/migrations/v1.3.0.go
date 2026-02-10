package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_3_0 adds organization_domains for auto-adding contacts by email domain.
func V1_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organization_domains (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			organization_id INT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE ON UPDATE CASCADE,
			domain TEXT NOT NULL,
			CONSTRAINT constraint_organization_domains_on_organization_id_and_domain UNIQUE (organization_id, domain),
			CONSTRAINT constraint_organization_domains_on_domain CHECK (length(domain) >= 2 AND length(domain) <= 253)
		);
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS index_organization_domains_on_domain ON organization_domains(domain);`)
	if err != nil {
		return err
	}
	_ = fs
	_ = ko
	return nil
}
