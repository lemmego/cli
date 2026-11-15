package migrations

import (
	"database/sql"
	"fmt"

	"github.com/lemmego/api/db"
	"github.com/lemmego/migration"
	"github.com/lemmego/notify/channel/database/sqlstore"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20261110000000",
		Up:      mig_20261110000000_create_notifications_table_up,
		Down:    mig_20261110000000_create_notifications_table_down,
	})
}

// The notifications table, for notify's in-app channel.
//
// The DDL comes from the module rather than being written out here, and that
// is the point: the store issues fixed SQL against fixed column names, and a
// second copy of the schema in this file is a second thing to keep in step.
// It drifts the first time the module adds a column, and the symptom is a
// query failing against a table that looks right.
//
// The dialect is a constant because the CLI knew it at generation time — this
// file was generated for {{.SQLDriver}}. The module ships statements for
// Postgres and SQLite and refuses anything else by design: an untested
// dialect string is worse than an honest refusal, because it looks like
// support right up to the point where somebody's JSON column or partial
// index fails inside their migration.
func mig_20261110000000_create_notifications_table_up(tx *sql.Tx) error {
	statements := sqlstore.SchemaStatements(
		{{- if eq .SQLDriver "postgres"}}
		db.Postgres,
		{{- else}}
		db.SQLite,
		{{- end}}
		sqlstore.DefaultTable,
	)
	if len(statements) == 0 {
		return fmt.Errorf("notify ships no schema for this dialect")
	}

	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("creating the notifications table: %w", err)
		}
	}
	return nil
}

func mig_20261110000000_create_notifications_table_down(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS " + sqlstore.DefaultTable)
	return err
}
