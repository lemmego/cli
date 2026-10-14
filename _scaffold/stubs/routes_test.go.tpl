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
// It **skips** when APPTEST_POSTGRES_DSN is unset rather than failing, which
// is the right default and also a trap worth knowing about: a suite that
// skips and a suite that passes are the same exit code. If `go test ./...`
// finishes suspiciously fast, that is why. Set the variable:
//
//	export APPTEST_POSTGRES_DSN="postgres://you@127.0.0.1:5432/postgres?sslmode=disable"
//
// Point it at any database on the server you want the scratch ones created
// on; this one is only used to issue CREATE DATABASE.
func TestTheApplicationBootsWithEveryProvider(t *testing.T) {
	a := apptest.New(t,
		apptest.WithConfig(config.GetAll()),
		apptest.WithProviders(bootstrap.LoadProviders()...),
		apptest.WithRoutes(bootstrap.LoadRoutes()...),
		apptest.WithPostgres(),
	)

	// Through the real middleware chain this time — the session wrapper, the
	// request logger, CSRF — rather than the bare router above.
	a.Get("/api/ping").Send().
		AssertStatus(200).
		AssertJSONPath("message", "pong")
}
