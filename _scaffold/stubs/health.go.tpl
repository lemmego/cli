package routes

import (
{{- if .HasDatabase}}
	"context"
	"errors"
	"time"

{{end}}
	"github.com/lemmego/api/app"
{{- if .HasDatabase}}
	"github.com/lemmego/api/db"
{{- end}}
)

// HealthRoutes exposes what the application knows about its own state.
//
// Three endpoints and not one, because they answer different questions and
// are consumed by different things:
//
//   - /livez says the process is running. An orchestrator restarts the
//     container when this fails, so it must not depend on anything external
//     — a liveness probe that fails when the database blinks restarts every
//     instance of a healthy application.
//   - /readyz says this instance can serve traffic. A load balancer takes it
//     out of rotation when this fails, so it checks the critical
//     dependencies and nothing else.
//   - /health is the full picture, for a person or a scraper. It reports
//     every check including the non-critical ones.
//
// Every container platform wants a path to poll, and api ships the handlers
// but registers no routes — so without this file a deployed project has
// nowhere to point a health check.
func HealthRoutes(a app.App) {
	checker := a.GetHealthChecker()
	if checker == nil {
		return
	}

	registerHealthChecks(a, checker)

	r := a.Router()
	// The health checker's handlers write their own JSON, so they are called
	// directly rather than through a response helper that would negotiate
	// content and wrap the body.
	r.Get("/health", func(c app.Context) error {
		checker.HealthCheckHandler()(c.ResponseWriter(), c.Request())
		return nil
	})
	r.Get("/livez", func(c app.Context) error {
		checker.LivenessHandler()(c.ResponseWriter(), c.Request())
		return nil
	})
	r.Get("/readyz", func(c app.Context) error {
		checker.ReadinessHandler()(c.ResponseWriter(), c.Request())
		return nil
	})
}

// registerHealthChecks decides what this application is asked about itself.
//
// Without at least one check the endpoints answer "healthy" having verified
// nothing, which is worse than not having them: it is a green light wired to
// no sensor.
func registerHealthChecks(a app.App, checker *app.HealthChecker) {
	// The filesystem and the session store, from the framework.
	checker.RegisterDefaultChecks()
{{- if .HasDatabase}}

	// The database. Critical, because nothing this application serves works
	// without it, so an instance that cannot reach it should be taken out of
	// rotation rather than left answering errors.
	checker.RegisterCheck("database", app.HealthCheck{
		Name:        "database",
		Description: "The application's database answers",
		Critical:    true,
		Timeout:     3 * time.Second,
		CheckFunc: func(ctx context.Context) error {
			// db.Resolve rather than the ORM's own handle, so this check is
			// the same whichever connector the project was generated with.
			conn, ok := db.Resolve(a)
			if !ok || conn == nil {
				return errors.New("no database is registered")
			}
			pool := conn.SQLDB()
			if pool == nil {
				return errors.New("the registered connection has no pool")
			}
			// A real round trip. A pool holding a connection object and a
			// server that has gone away look identical until something asks.
			return pool.PingContext(ctx)
		},
	})
{{- end}}
}
