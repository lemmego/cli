package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	apidb "github.com/lemmego/api/db"
	"github.com/lemmego/auth"
	"github.com/lemmego/orm"
	"github.com/lemmego/tenancy"
	"github.com/lemmego/tenancy/hostname"
	tenancyprovider "github.com/lemmego/tenancy/provider"
	"github.com/lemmego/tenancy/rls"
	"github.com/lemmego/tenancy/sqlstore"

	"github.com/lemmego/lemmego/internal/models"
)

// rlsConfig is the row-level security setting a policy reads and the column
// every scoped table carries.
//
// One place for it, because the migration that writes a policy and the
// provider that sets the value have to agree. A mismatch means the policy
// reads a setting nothing ever sets, which denies everything or allows
// everything depending on how the policy was written — and neither failure
// looks like a configuration mistake.
func rlsConfig() rls.Config {
	return rls.Config{}
}

// TenancyProvider is the module's provider with this application's connection
// filled in.
//
// A wrapper rather than a literal, because the registry and the handle both
// need a live pool and the provider list is written before the container
// exists. Embedding the module's provider keeps its own request-path wiring
// rather than reimplementing it.
type TenancyProvider struct {
	*tenancyprovider.Provider
}

// Tenancy resolves which tenant each request is about and opens a transaction
// bound to it.
//
// A tenant is read off the host — company.example.test reaches the company
// tenant — and a header is accepted too. The distinction that matters: a
// header is client-supplied, so naming a tenant is not the same as being
// allowed to see it. Authorisation is the Principal callback plus the
// membership query below, never the fact that a name arrived.
//
// After the database connector and after auth: the registry reads the tenants
// table, and the principal is whoever the auth provider resolved.
func Tenancy() app.Provider {
	return &TenancyProvider{
		Provider: &tenancyprovider.Provider{
			Config: tenancy.Config{
				Hostnames: hostname.Config{
					// Every host under this domain resolves to a tenant
					// label. A zero value is refused at boot rather than
					// defaulted: resolving every host to one surface because
					// nobody configured a domain is not a sensible default.
					AppDomain: config.MustEnv("TENANCY_APP_DOMAIN", "example.test"),
				},
			},

			RLS: rlsConfig(),

			// Who is making the request. Supplied here because tenancy must
			// not learn what a session is.
			Principal: func(c app.Context) (tenancy.Principal, bool) {
				user, ok := auth.UserAs[*models.User](c)
				if !ok {
					return nil, false
				}
				return user, true
			},

			// Paths that never name a tenant. Resolution is cheap but not
			// free, and a transaction per request for a static file is waste.
			SkipPrefixes: []string{"/assets", "/public", "/health", "/livez", "/readyz"},
		},
	}
}

// Provide fills in the connection-dependent fields, runs the two row-level
// security checks, and then boots the module's provider.
func (p *TenancyProvider) Provide(a app.App) error {
	conn, ok := apidb.Resolve(a)
	if !ok {
		return fmt.Errorf("tenancy: no database connection; list this provider after the connector")
	}
	pool := conn.SQLDB()

	ormDB, ok := app.Lookup[*orm.DB](a)
	if !ok {
		return fmt.Errorf("tenancy: no *orm.DB registered; tenant transactions open on it")
	}

	p.Registry = sqlstore.New(pool, sqlstore.Config{
		Placeholder: sqlstore.PostgresPlaceholder,
		Members:     memberOfTenant,
	})
	p.DB = ormDB

	if err := p.checkIsolation(a, pool); err != nil {
		return err
	}

	return p.Provider.Provide(a)
}

// checkIsolation runs the two checks that decide whether the database is
// actually enforcing anything.
//
//   - A superuser bypasses row-level security entirely, and FORCE does not
//     change that — FORCE only subjects the table's owner. A development
//     PostgreSQL role is usually a superuser, so without this check the
//     policies are inert locally and an isolation test passes while proving
//     nothing.
//   - A connection pooler in *statement* mode makes the tenant setting
//     session-level rather than transaction-local, which leaks the tenant
//     across requests under load.
//
// Fatal in production, a warning elsewhere: a developer's local role usually
// is a superuser, and refusing to boot would mean the first thing a new
// project does is stop.
func (p *TenancyProvider) checkIsolation(a app.App, pool *sql.DB) error {
	ctx := context.Background()

	checks := []struct {
		what string
		run  func() error
	}{
		{"the application role bypasses row-level security", func() error {
			return rls.AssertNotPrivileged(ctx, pool)
		}},
		{"the tenant setting is not transaction-local", func() error {
			return rlsConfig().SelfTest(ctx, pool)
		}},
	}

	for _, check := range checks {
		err := check.run()
		if err == nil {
			continue
		}
		if a.InProduction() {
			return fmt.Errorf("tenancy: %s: %w", check.what, err)
		}
		slog.Warn("tenancy: "+check.what,
			"error", err,
			"effect", "the database is not enforcing tenant isolation here")
	}
	return nil
}

// memberOfTenant answers whether this principal belongs to this tenant.
//
// The transaction handed in is already scoped to that tenant, so this query
// needs no tenant predicate of its own and cannot see another tenant's rows.
func memberOfTenant(ctx context.Context, tx *sql.Tx, tenantID string, principal tenancy.Principal) (bool, error) {
	// Only this population lives in tenant_users. Another one is a different
	// table and a different query, and answering false is the safe default
	// rather than guessing.
	if principal.PrincipalKind() != "user" {
		return false, nil
	}

	var exists bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (
		    SELECT 1 FROM tenant_users
		    WHERE tenant_id = $1 AND user_id = $2
		 )`,
		tenantID, principal.PrincipalKey(),
	).Scan(&exists)
	return exists, err
}

// TenantTx is the handle a repository should use inside a request.
//
// It is the transaction the tenant is bound to: queries run on it are scoped,
// and queries run on the bare handle are not. A repository that reaches for
// the handle instead of this is how a scoped read becomes an unscoped one.
func TenantTx(c app.Context) (*orm.Tx, bool) {
	return tenancyprovider.Tx(c.RequestContext())
}
