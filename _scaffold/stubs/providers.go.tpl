package bootstrap

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/providers/fs"
	"github.com/lemmego/api/providers/session"
	{{- if .EnableAuth}}
	"github.com/lemmego/api/config"
	"github.com/lemmego/auth"
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
	return []app.Provider{
		&fs.Provider{},
		&session.Provider{},
		{{- if .UseORMConnector}}
		&ormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseGormConnector}}
		&gormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseBunConnector}}
		&bunconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .HasCache}}
		&cache.Provider{},
		{{- end}}
		{{- if .HasQueue}}
		&queue.Provider{},
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
}
