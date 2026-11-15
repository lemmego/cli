package plugins

import (
	"crypto/subtle"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/observe"
	observeprovider "github.com/lemmego/observe/provider"
)

// Observe records what one request did — every query, log line, cache
// lookup, job and email it caused, in order, with timings.
//
// Registered after the database connector whose pool it borrows and before
// anything whose queries are worth seeing.
//
// **Off unless OBSERVE_ENABLED says otherwise**, and that default is not
// timidity: an entry holds request headers, bound parameters and every
// statement the application ran, which is session ids, bearer tokens and
// whatever was in the body. Telescope exposed in production is a known
// breach vector and this records the same things.
//
// With a database it keeps entries in a table, which needs the migration
// published first — it is not created at boot, because creating tables
// silently would put tables in somebody's database that nothing in their
// project mentions:
//
//	lemmego run publish --tags=observe-migrations
//	go build ./...          # the published migration has to be compiled in
//	lemmego run migrate up
//
// That second step is the one people miss: the migration was written by a
// running binary that does not yet contain it. With no database it keeps
// them in memory, which is only right in one process — `lemmego run
// mcp:serve` is a different process and would see none of them.
//
// **Queries are recorded only through the instrumented driver**, which is
// registered in observe_driver.go and handed to the connector in
// bootstrap/providers.go. That wiring is done for the Lemmego ORM, whose
// connector takes a driver name. It is not done for GORM or Bun: GORM is
// given a dialector that builds its own pool, and Bun opens a
// driver.Connector directly, so neither goes through the registry a driver
// name selects. Both can be instrumented by handing them a *sql.DB opened
// on ObservedDriver — and until that is done, observe records requests,
// logs, cache operations and jobs for them, but no SQL.
func Observe() app.Provider {
	return &observeprovider.Provider{
		DashboardAuth: dashboardToken,

		// The fraction of requests kept, decided once where the correlation
		// id is minted rather than per entry — so a request that will be
		// discarded does not first pay for redacting and rendering every
		// bound parameter. That ordering is the difference between sampling
		// saving a third of the work and saving almost all of it.
		//
		// Exceptions and errors bypass it. A sampled-out request that panics
		// is the one you most need, and a tool that drops nine out of ten
		// crashes is not a telemetry tool.
		Sampler: observe.SampleRequests(config.MustEnv("OBSERVE_SAMPLE", 1.0)),
	}
}

// dashboardToken gates the dashboard on a shared secret.
//
// A token rather than a staff check, because this application may have no
// authentication at all and the dashboard still needs a door. If it does
// have authentication, prefer the user: read the signed-in account here and
// compare it against a list of admins, the way the queue dashboard does.
//
// **An unset token means no access, not open access.** That is the whole
// point of the default: leaving it nil or empty is not "no authentication",
// it is "nobody gets in" — and the alternative mistake is a dashboard that
// serves every statement this application has run to anyone who finds the
// URL.
func dashboardToken(c app.Context) bool {
	want := config.MustEnv("OBSERVE_TOKEN", "")
	if want == "" {
		return false
	}

	// Constant time, so the comparison does not leak the token one byte at a
	// time to somebody who can measure it.
	got := c.Request().Header.Get("X-Observe-Token")
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
