package main

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	_ "github.com/lemmego/api/logger"
	"github.com/lemmego/lemmego/bootstrap"
	_ "github.com/lemmego/lemmego/internal/configs"
	_ "github.com/lemmego/lemmego/internal/migrations"
)

func main() {
	webApp := app.Configure(app.WithConfig(config.GetAll()))

	webApp.WithRoutes(bootstrap.LoadRoutes()).
		WithHTTPMiddlewares(bootstrap.LoadHTTPMiddlewares()).
		WithMiddlewares(bootstrap.LoadMiddlewares()).
		WithCommands(bootstrap.LoadCommands()).
		WithProviders(bootstrap.LoadProviders()).
		WithErrMap(bootstrap.LoadErrMap())

	webApp.Run()
}
