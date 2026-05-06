package repos

import (
	"github.com/uptrace/bun"

	"github.com/lemmego/api/app"
)

func getDB(a app.App) *bun.DB {
	return app.Get[*bun.DB](a)
}
