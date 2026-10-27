package bootstrap

import (
	{{- if and .HasQueue .EnableAuth}}
	"slices"
	"strings"

	{{- end}}
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/logger"
	"github.com/lemmego/api/providers/fs"
	"github.com/lemmego/api/providers/session"
	{{- if eq .FilesystemDisk "s3"}}
	// The S3 driver, for its side effect. fsys serves local from its root and
	// every cloud driver from a subpackage, so a deployment links only the
	// storage it uses. Without this the s3 disk fails when it is first
	// opened, not at compile time.
	_ "github.com/lemmego/fsys/s3"
	{{- end}}
	{{- if and .UseMCP (not .EnableAuth)}}
	"github.com/lemmego/api/config"
	{{- end}}
	{{- if .EnableAuth}}
	"github.com/lemmego/api/config"
	"github.com/lemmego/auth"
	{{- if .HasQueue}}
	"github.com/lemmego/lemmego/internal/models"
	{{- end}}
	"github.com/lemmego/lemmego/internal/repos"
	{{- end}}
	{{- if .HasCache}}
	"github.com/lemmego/cache"
	// Only the store this project uses is registered, so a project that does
	// not cache in Redis does not link a Redis client. Import
	// github.com/lemmego/cache/drivers instead to register all four.
	_ "github.com/lemmego/cache/store/{{.CacheDriver}}"
	{{- end}}
	{{- if .HasQueue}}
	"github.com/lemmego/queue"
	{{- end}}
	{{- if .UseMCP}}
	lemmegomcp "github.com/lemmego/mcp"
	{{- end}}
	{{- if .InertiaProvider}}
	"github.com/lemmego/inertia"
	{{- end}}
	{{- if .UseORMConnector}}
	"github.com/lemmego/ormconnector"
	{{- end}}
	{{- if .UseGormConnector}}
	"github.com/lemmego/gormconnector"
	{{- end}}
	{{- if .UseBunConnector}}
	"github.com/lemmego/bunconnector"
	{{- end}}
)

// LoadProviders lists the providers the application boots, in order.
//
// Order matters. Providers run one at a time, in this order, and a provider
// can only resolve services the ones before it registered. The database
// connector therefore comes first among the providers that own resources:
// it publishes the connection under db.Connection, which is how any package
// that needs to store something finds this application's pool instead of
// opening a second one of its own.
func LoadProviders() []app.Provider {
	providers := []app.Provider{
		// First, and for a reason worth keeping: every provider after this
		// one logs while it boots, and a record emitted before this runs goes
		// to slog's bare default instead of the file, format and level this
		// application configured. Set LOG_CHANNEL and the rest in .env, or
		// fix something in code with &logger.Provider{Options: ...}.
		&logger.Provider{},
		// The public disk is served in full at /storage/, and the default
		// disk is not served at all — see internal/configs/filesystems.go
		// for the layout and why the split rather than an allowlist.
		//
		// Uploads reach it with c.UploadOn("public", field, dir, name). The
		// default disk is for anything that should not be readable by
		// anybody with the URL.
		&fs.Provider{PublicDisk: "public", PublicEntireDisk: true},
		{{- if .UseORMConnector}}
		&ormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseGormConnector}}
		&gormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseBunConnector}}
		&bunconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		// After the connectors, and that ordering is load-bearing rather than
		// tidy: SESSION_DRIVER=database resolves the application's connection
		// through db.Resolve, which finds nothing if this runs first. The
		// provider refuses to boot in that case instead of falling back,
		// because the fallback is an application that looks healthy and signs
		// everybody out.
		&session.Provider{},
		{{- if .HasCache}}
		&cache.Provider{},
		{{- end}}
		{{- if .HasQueue}}
		&queue.Provider{
			{{- if .EnableAuth}}
			// The queue dashboard can retry, cancel and delete jobs, so it
			// stays shut until this says otherwise. Add yourself to
			// TASKER_ADMINS to get in.
			DashboardAuth: func(c app.Context) bool {
				// Check first. The dashboard is mounted as a raw handler, so
				// none of the route middleware ran for this request and
				// nothing has looked at the cookie yet; without this the
				// user is always absent and the answer is always no.
				if err := auth.Check(c); err != nil {
					return false
				}
				user, ok := auth.UserAs[*models.User](c)
				return ok && slices.Contains(taskerAdmins(), user.Email)
			},
			{{- else}}
			// This project has no authentication, so there is nobody to
			// recognise and the dashboard stays closed. Set DashboardAuth to
			// a predicate of your own — a shared secret header, say — to
			// open it.
			{{- end}}
		},
		{{- end}}
		{{- if .InertiaProvider}}
		&inertia.Provider{
			Options: []inertia.Option{
				inertia.WithSSR(),
			},
		},
		{{- end}}
		{{- if .EnableAuth}}
		&auth.Provider{
			Opts: &auth.Opts{
				// A credential carries an id, never a copy of the user. This
				// is the one place that turns that id back into a row, so
				// every request resolves the same concrete type no matter
				// which credential arrived — a session cookie, a bearer
				// token, or an OAuth2 access token. Handlers read it with
				// auth.UserAs[*models.User](c).
				//
				// It costs one indexed lookup per authenticated request, and
				// buys the thing a token cannot give you: the row as it is
				// now, not as it was when the token was minted. A user
				// deactivated a minute ago is refused on the next request
				// rather than at expiry.
				//
				// The loader takes app.Context because LoadProviders runs
				// before the container exists — there is no app.App to close
				// over here. c.App() resolves it per request.
				UserLoader: func(c app.Context, id string) (any, error) {
					{{- if .UseGPA}}
					return repos.User().FindByID(c.RequestContext(), id)
					{{- else}}
					return repos.User(c.App()).FindByID(c.RequestContext(), id)
					{{- end}}
				},
				{{- if eq .Preset "mvc"}}
				// A browser client gets a session: logging out actually
				// revokes, because the server holds the record. JwtSecret is
				// still set, so /api/* keeps taking bearer tokens.
				DisableSession: false,
				{{- else}}
				// An API has no cookie jar to renew, so there is no session
				// to keep. Note that a bearer token cannot be revoked before
				// it expires.
				DisableSession: true,
				{{- end}}
				JwtSecret: config.MustEnv("JWT_SECRET", config.MustEnv("APP_KEY", "")),
			},
		},
		{{- end}}
	}
	{{- if .UseMCP}}

	// The MCP server, last and only outside production.
	//
	// Last, because it reports on everything the providers above register —
	// the database connection, the session store, the health checks — and a
	// provider listed before them would resolve nothing.
	//
	// Not in production, because it serves this application's configuration
	// and database to whatever starts it. It is stdio-only, so it is not
	// reachable over the network and this is not a hole being closed; it is a
	// development tool, and a production binary should not carry a command
	// that reads the database. Remove the guard if you want it everywhere.
	if config.MustEnv("APP_ENV", "development") != "production" {
		providers = append(providers, &lemmegomcp.Provider{})
	}
	{{- end}}

	return providers
}

{{- if and .HasQueue .EnableAuth}}

// taskerAdmins lists who may use the queue dashboard, read from
// TASKER_ADMINS as a comma-separated list of email addresses.
//
// It is empty by default, which closes the dashboard rather than opening it
// to every signed-in user: the dashboard retries, cancels and deletes jobs,
// and a queue is not something an ordinary account should be able to clear.
func taskerAdmins() []string {
	raw := config.MustEnv("TASKER_ADMINS", "")
	if raw == "" {
		return nil
	}
	admins := strings.Split(raw, ",")
	for i := range admins {
		admins[i] = strings.TrimSpace(admins[i])
	}
	return admins
}
{{- end}}
