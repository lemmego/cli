package routes

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
	tenancyprovider "github.com/lemmego/tenancy/provider"
)

// TenancyRoutes is the worked example: one route that reports which tenant
// the request resolved to.
//
// Behind auth.Protected, because the binding comes from the caller's
// membership and an anonymous request has none.
//
// Reach it on a tenant host — company.example.test:1490/tenancy/demo with
// TENANCY_APP_DOMAIN=example.test — or with the X-Tenant-Slug header. Both
// only *name* a tenant; whether the caller sees it is the membership row.
func TenancyRoutes(a app.App) {
	a.Router().Get("/tenancy/demo", auth.Protected, tenancyDemo)
}

func tenancyDemo(c app.Context) error {
	tenant, ok := tenancyprovider.Current(c.RequestContext())
	if !ok {
		// No tenant bound: the host matched no tenant, or the signed-in user
		// is not a member of the one named. Those are the same answer on
		// purpose — a distinct "forbidden" would confirm which companies are
		// customers.
		return c.SetStatus(404).JSON(app.M{
			"error": "no tenant for this request",
		})
	}

	// The transaction every scoped query should run on. A query on the bare
	// handle is not scoped, whatever this request resolved to.
	_, hasTx := tenancyprovider.Tx(c.RequestContext())

	return c.JSON(app.M{
		"tenant":        tenant.Slug,
		"name":          tenant.Name,
		"id":            tenant.ID,
		"has_tenant_tx": hasTx,
	})
}
