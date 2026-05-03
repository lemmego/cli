package bootstrap

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/routes"
)

func LoadRoutes() []app.RouteCallback {
	return []app.RouteCallback{
		{{- if eq .Preset "mvc"}}
		routes.WebRoutes,
		{{- end}}
		routes.ApiRoutes,
	}
}
