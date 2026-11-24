package migrations

import (
	"database/sql"
	"fmt"

	"github.com/lemmego/migration"
	"github.com/lemmego/rbac/rbacstore"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20261113000000",
		Up:      mig_20261113000000_create_rbac_tables_up,
		Down:    mig_20261113000000_create_rbac_tables_down,
	})
}

// The roles, grants and assignments tables.
//
// The DDL comes from the module rather than being written out here: the store
// issues fixed SQL against fixed column names, and a second copy of the
// schema in this file is a second thing to keep in step. It drifts the first
// time the module adds a column, and the symptom is a query failing against a
// table that looks right.
//
// A zero SchemaConfig takes every default — text scope, text subject, no
// foreign keys — which is what plugins.RbacStore() is built with, so the two
// cannot disagree. Set ScopeReferences and SubjectReferences when your scopes
// and subjects are rows you want the database to enforce against; the
// subject reference is composite and names the scope column too.
func mig_20261113000000_create_rbac_tables_up(tx *sql.Tx) error {
	for _, statement := range rbacstore.SchemaStatements(rbacstore.SchemaConfig{}) {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("creating the rbac tables: %w", err)
		}
	}
	return nil
}

func mig_20261113000000_create_rbac_tables_down(tx *sql.Tx) error {
	for _, statement := range rbacstore.DropStatements(rbacstore.SchemaConfig{}) {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("dropping the rbac tables: %w", err)
		}
	}
	return nil
}
