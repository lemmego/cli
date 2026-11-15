package routes

{{- /*
The import conditions below mirror the if/else-if/else in WebRoutes exactly,
and they have to. They used to be three independent blocks each importing
api/app, so the two templ+inertia presets — where FrontendHasTempl and
InertiaProvider are both true — emitted "github.com/lemmego/api/app" twice and
produced a file that could not compile, with templ and templates imported and
unused on top of it.

formatGoFiles did collapse the duplicate api/app import, so the compiler never
reported a redeclaration — it reported templates and templ "imported and not
used", which is the same broken project by a quieter route. The offline checks
missed it because parser.ParseFile accepts a duplicate import (a type error,
not a parse error) and the unused-import check keys on the identifier, so a
single app. reference marked every copy used. Only a compiler sees this, which
is the argument for the build gate in generated_auth_identity_test.go covering
every frontend preset rather than for adding more parse-level checks.
*/}}
import (
	"github.com/lemmego/api/app"
	{{- if .HasModule "seo"}}
	"github.com/lemmego/seo"
	{{- end}}
	{{- if .InertiaProvider}}
	"github.com/lemmego/inertia"
	{{- else if .FrontendHasTempl}}
	"github.com/lemmego/lemmego/templates"
	"github.com/lemmego/templ"
	{{- else}}
	"github.com/lemmego/api/res"
	{{- end}}
)

func WebRoutes(a app.App) {
	r := a.Router()
	r.Get("/{$}", func(c app.Context) error {
		{{- if .HasModule "seo"}}
		// Called immediately before responding, which is where it belongs:
		// Share renders the head in Go and publishes it as template data, so
		// the title, the canonical URL and any structured data survive a
		// server-rendering failure — the kind that answers 200 and looks
		// perfect to a human while a crawler receives an empty container.
		//
		// seo.Middleware already shares the site-wide defaults for every
		// page, so this is what adds the part only this page knows.
		seo.Share(c, seo.Meta{
			Title:       "Home",
			Description: "A new Lemmego application.",
			Canonical:   c.Request().URL.Path,
		})
		{{- end}}
		{{- if .InertiaProvider}}
		return inertia.Respond(c, "{{if .FrontendHasVue}}IndexVue{{else}}IndexReact{{end}}", nil)
		{{- else if .FrontendHasTempl}}
		return templ.Respond(c, templates.BaseLayout(templates.Index()))
		{{- else}}
		return c.Render(res.NewTemplate(c, "index.page.gohtml"))
		{{- end}}
	})
}
