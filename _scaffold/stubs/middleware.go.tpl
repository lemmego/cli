package bootstrap

import (
	"github.com/lemmego/api/app"
	{{- if eq .Preset "mvc"}}
	"github.com/lemmego/api/middleware"
	{{- end}}
	{{- if .HasModule "seo"}}
	"github.com/lemmego/seo"
	{{- end}}
)

func LoadMiddlewares() []app.Handler {
	return []app.Handler{
		{{- if eq .Preset "mvc"}}
		middleware.VerifyCSRF(&middleware.CSRFOpts{ExcludePatterns: []string{"/api/.*"}}),
		{{- end}}
		{{- if .HasModule "seo"}}
		// The site-wide defaults, shared for every page, so a handler that
		// renders without calling seo.Share still emits a title, a canonical
		// URL and the right robots directive rather than nothing. Without
		// this, forgetting one call is a page that is silently unindexable.
		seo.Middleware(),
		{{- end}}
	}
}
