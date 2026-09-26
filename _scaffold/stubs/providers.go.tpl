package bootstrap

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/providers/fs"
	"github.com/lemmego/api/providers/session"
	{{- if .EnableAuth}}
	"github.com/lemmego/api/config"
	"github.com/lemmego/auth"
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

func LoadProviders() []app.Provider {
	return []app.Provider{
		&fs.Provider{},
		&session.Provider{},
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
		{{- if .UseORMConnector}}
		&ormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseGormConnector}}
		&gormconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .UseBunConnector}}
		&bunconnector.Provider{{if .UseGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if .EnableAuth}}
		&auth.Provider{
			Opts: &auth.Opts{
				DisableSession: true,
				JwtSecret:      config.MustEnv("JWT_SECRET", config.MustEnv("APP_KEY", "")),
			},
		},
		{{- end}}
	}
}
