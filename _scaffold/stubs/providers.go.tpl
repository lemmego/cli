package bootstrap

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/providers/fs"
	"github.com/lemmego/api/providers/session"
	{{- if .EnableAuth}}
	"github.com/lemmego/api/config"
	{{- end}}
	"github.com/lemmego/cache"
	// Registers the memory, file, redis and null cache stores. Import only the
	// ones you use instead if you want to keep a Redis client out of the
	// binary.
	_ "github.com/lemmego/cache/drivers"
	"github.com/lemmego/queue"
	{{- if .InertiaProvider}}
	"github.com/lemmego/inertia"
	{{- end}}
	{{- if eq .ORM "orm"}}
	"github.com/lemmego/ormconnector"
	{{- end}}
	{{- if eq .ORM "gorm"}}
	"github.com/lemmego/gormconnector"
	{{- end}}
	{{- if eq .ORM "bun"}}
	"github.com/lemmego/bunconnector"
	{{- end}}
	{{- if .EnableAuth}}
	"github.com/lemmego/auth"
	{{- end}}
)

func LoadProviders() []app.Provider {
	return []app.Provider{
		&fs.Provider{},
		&session.Provider{},
		&cache.Provider{},
		&queue.Provider{},
		{{- if .InertiaProvider}}
		&inertia.Provider{
			Options: []inertia.Option{
				inertia.WithSSR(),
			},
		},
		{{- end}}
		{{- if eq .ORM "orm"}}
		&ormconnector.Provider{{if .EnableGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if eq .ORM "gorm"}}
		&gormconnector.Provider{{if .EnableGPA}}{UseGPA: true}{{else}}{}{{end}},
		{{- end}}
		{{- if eq .ORM "bun"}}
		&bunconnector.Provider{{if .EnableGPA}}{UseGPA: true}{{else}}{}{{end}},
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
