package main

import (
	"github.com/lemmego/api/app"
	_ "github.com/lemmego/api/logger"
	"github.com/lemmego/lemmego/bootstrap"
	_ "github.com/lemmego/lemmego/internal/configs"
	{{- if .EnableAuth}}
	_ "github.com/lemmego/lemmego/internal/migrations"
	{{- end}}
)

func main() {
	webApp := app.Configure()

	webApp.WithRoutes(bootstrap.LoadRoutes()).
		WithHTTPMiddlewares(bootstrap.LoadHTTPMiddlewares()).
		WithMiddlewares(bootstrap.LoadMiddlewares()).
		WithCommands(bootstrap.LoadCommands()).
		WithProviders(bootstrap.LoadProviders())

	webApp.Run()
}
