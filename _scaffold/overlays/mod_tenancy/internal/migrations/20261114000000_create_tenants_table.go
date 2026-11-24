package migrations

import (
	"database/sql"
	"fmt"

	"github.com/lemmego/migration"
)

func init() {
	migration.GetMigrator().AddMigration(&migration.Migration{
		Version: "20261114000000",
		Up:      mig_20261114000000_create_tenants_table_up,
		Down:    mig_20261114000000_create_tenants_table_down,
	})
}

// The tenants registry and the membership table.
//
// Raw SQL rather than the schema builder, because the column names are the
// ones sqlstore queries by default — id, slug, name, status, deleted_at — and
// the id is a uuid because the store casts it as one. Renaming any of them
// means passing a matching sqlstore.Config in internal/plugins/tenancy.go.
//
// Row-level security is not enabled on either of these. The registry resolves
// a tenant *before* any tenant is bound, so a policy on tenants would refuse
// the lookup that decides which tenant to bind. Policies belong on your own
// scoped tables; see the commented example at the bottom of this file.
var tenantsUp = []string{
	`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,

	`CREATE TABLE tenants (
	    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
	    slug       text        NOT NULL,
	    name       text        NOT NULL,
	    status     text        NOT NULL DEFAULT 'active',
	    created_at timestamptz NOT NULL DEFAULT now(),
	    updated_at timestamptz NOT NULL DEFAULT now(),
	    deleted_at timestamptz
	 )`,

	// Unique among live rows only, so an archived tenant's slug can be
	// reused without a partial-index-free workaround.
	`CREATE UNIQUE INDEX tenants_slug_live ON tenants (lower(slug)) WHERE deleted_at IS NULL`,

	// Which users belong to which tenant. The membership query in
	// internal/plugins/tenancy.go reads exactly this.
	//
	// A caller who names a tenant they are not in gets the same error as a
	// tenant that does not exist, so this table is what separates "may see
	// it" from "typed its name".
	`CREATE TABLE tenant_users (
	    tenant_id  uuid        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
	    user_id    text        NOT NULL,
	    created_at timestamptz NOT NULL DEFAULT now(),
	    PRIMARY KEY (tenant_id, user_id)
	 )`,

	`CREATE INDEX tenant_users_user ON tenant_users (user_id)`,
}

// Enabling row-level security on one of your own tables looks like this. Add
// it to the migration that creates that table, not here:
//
//	ALTER TABLE notes ADD COLUMN tenant_id uuid NOT NULL
//	    REFERENCES tenants (id) ON DELETE CASCADE;
//	ALTER TABLE notes ENABLE ROW LEVEL SECURITY;
//	ALTER TABLE notes FORCE ROW LEVEL SECURITY;
//	CREATE POLICY notes_tenant ON notes
//	    USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
//
// FORCE matters because without it the table's owner is exempt, and the
// application role is often the owner. It does not help against a superuser:
// nothing does, which is what the boot check in the plugin is for.
var tenantsDown = []string{
	`DROP TABLE IF EXISTS tenant_users`,
	`DROP TABLE IF EXISTS tenants`,
}

func mig_20261114000000_create_tenants_table_up(tx *sql.Tx) error {
	for _, statement := range tenantsUp {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("creating the tenants tables: %w", err)
		}
	}
	return nil
}

func mig_20261114000000_create_tenants_table_down(tx *sql.Tx) error {
	for _, statement := range tenantsDown {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("dropping the tenants tables: %w", err)
		}
	}
	return nil
}
