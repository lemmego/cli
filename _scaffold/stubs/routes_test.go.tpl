package routes_test

import (
	"testing"

	"github.com/lemmego/api/config"
	"github.com/lemmego/apptest"

	"{{.ModuleName}}/bootstrap"
	_ "{{.ModuleName}}/internal/configs"
)

// The application under test is the application that ships.
//
// apptest.New goes through app.Configure and engine.Handler — the same
// sequence Run performs before it listens. Providers run, middleware
// installs, routes mount. A harness that reassembled those pieces itself
// would drift from production, and a test that passes against a drifted
// harness is worse than no test.

// Routes and nothing else: no providers, so no database, so this runs
// anywhere. Useful for routing, redirects and status codes.
func TestTheAPIAnswersAPing(t *testing.T) {
	a := apptest.New(t,
		apptest.WithRoutes(bootstrap.LoadRoutes()...),
	)

	a.Get("/api/ping").Send().
		AssertStatus(200).
		AssertJSONPath("message", "pong")
}

// The whole stack, against a real scratch database that is created, migrated
// and dropped around the test.
//
// The server comes from this application's own configuration, which under a
// test binary means .env.test. There is no separate environment variable
// naming a database, deliberately: two ways to say which server to use is two
// things that can disagree.
//
// It **fails rather than skipping** when that server is unreachable. Skipping
// looks kinder and costs more: a suite that skips and a suite that passes are
// the same exit code, so the only way to tell them apart is the clock. A test
// that needs a database and has none has not passed.
//
// So make sure .env.test points at a Postgres you are running. The scratch
// databases are created on it and dropped when each test ends.
func TestTheApplicationBootsWithEveryProvider(t *testing.T) {
	a := apptest.New(t,
		apptest.WithConfig(config.GetAll()),
		apptest.WithProviders(bootstrap.LoadProviders()...),
		apptest.WithRoutes(bootstrap.LoadRoutes()...),
		apptest.WithScratchDB(),
	)

	// Through the real middleware chain this time — the session wrapper, the
	// request logger, CSRF — rather than the bare router above.
	a.Get("/api/ping").Send().
		AssertStatus(200).
		AssertJSONPath("message", "pong")
}
