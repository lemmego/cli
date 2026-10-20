package migrations

import (
	"database/sql"

	"github.com/lemmego/migration"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20250903034100",
		Up:      mig_20250903034100_create_sessions_table_up,
		Down:    mig_20250903034100_create_sessions_table_down,
	})
}

// The sessions table, for SESSION_DRIVER=database.
//
// Written as raw SQL rather than through the schema builder because the schema
// is not this project's to choose: api/providers/session hands the connection
// to one of the scs SQL stores, and each issues fixed SQL against fixed column
// names and types. The builder would produce the house conventions — an id,
// timestamps — and a table the store cannot query.
//
// That is also why the types differ per database rather than being one
// portable definition: scs ships a separate store per dialect, and each reads
// `expiry` differently. This file was generated for {{.SQLDriver}}.
//
// The index is what keeps the store's periodic sweep of expired rows cheap;
// without it every sweep scans the table.
func mig_20250903034100_create_sessions_table_up(tx *sql.Tx) error {
	statements := []string{
{{- if eq .SQLDriver "postgres"}}
		`CREATE TABLE sessions (
			token  TEXT        NOT NULL PRIMARY KEY,
			data   BYTEA       NOT NULL,
			expiry TIMESTAMPTZ NOT NULL
		)`,
{{- else if eq .SQLDriver "mysql"}}
		`CREATE TABLE sessions (
			token  CHAR(43) COLLATE utf8mb4_bin NOT NULL PRIMARY KEY,
			data   BLOB         NOT NULL,
			expiry TIMESTAMP(6) NOT NULL
		)`,
{{- else}}
		`CREATE TABLE sessions (
			token  TEXT NOT NULL PRIMARY KEY,
			data   BLOB NOT NULL,
			expiry REAL NOT NULL
		)`,
{{- end}}
		`CREATE INDEX sessions_expiry_idx ON sessions (expiry)`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

// Dropping the table signs everybody out, which is the honest consequence of
// rolling this back: the sessions are the data.
func mig_20250903034100_create_sessions_table_down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS sessions`)
	return err
}
