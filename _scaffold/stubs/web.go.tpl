package routes

import (
	{{- if eq .Frontend "go_templates"}}
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/res"
	{{- end}}
	{{- if .FrontendHasTempl}}
	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/templates"
	"github.com/lemmego/templ"
	{{- end}}
	{{- if .InertiaProvider}}
	"github.com/lemmego/api/app"
	"github.com/lemmego/inertia"
	{{- end}}
)

func WebRoutes(a app.App) {
	r := a.Router()
	r.Get("/{$}", func(c app.Context) error {
		{{- if .InertiaProvider}}
		return inertia.Respond(c, "IndexReact", nil)
		{{- else if .FrontendHasTempl}}
		return templ.Respond(c, templates.BaseLayout(templates.Index()))
		{{- else}}
		return c.Render(res.NewTemplate(c, "index.page.gohtml"))
		{{- end}}
	})
}
