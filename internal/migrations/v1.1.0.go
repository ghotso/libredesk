package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V1_1_0 adds portal and organizations settings for existing installations.
func V1_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		INSERT INTO settings ("key", value) VALUES
			('app.portal_enabled', 'false'::jsonb),
			('app.portal_default_inbox_id', '0'::jsonb),
			('app.organizations_enabled', 'false'::jsonb)
		ON CONFLICT ("key") DO NOTHING;
	`)
	return err
}
