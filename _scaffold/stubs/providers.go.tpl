package bootstrap

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/providers/fs"
	"github.com/lemmego/api/providers/session"
	{{- if .InertiaProvider}}
	"github.com/lemmego/inertia"
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
		{{- if .InertiaProvider}}
		&inertia.Provider{},
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
				JwtSecret:      "a-long-long-secret",
			},
		},
		{{- end}}
	}
}
