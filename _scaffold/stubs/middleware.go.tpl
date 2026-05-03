package bootstrap

import (
	"github.com/lemmego/api/app"
	{{- if eq .Preset "mvc"}}
	"github.com/lemmego/api/middleware"
	{{- end}}
)

func LoadMiddlewares() []app.Handler {
	return []app.Handler{
		{{- if eq .Preset "mvc"}}
		middleware.VerifyCSRF(&middleware.CSRFOpts{ExcludePatterns: []string{"/api/.*"}}),
		{{- end}}
	}
}
