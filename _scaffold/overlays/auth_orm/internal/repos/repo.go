package repos

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/orm"
)

func getDB(a app.App) *orm.DB {
	return app.Get[*orm.DB](a)
}
