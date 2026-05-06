package repos

import (
	"gorm.io/gorm"

	"github.com/lemmego/api/app"
)

func getDB(a app.App) *gorm.DB {
	return app.Get[*gorm.DB](a)
}
