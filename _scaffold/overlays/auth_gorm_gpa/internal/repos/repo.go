package repos

import (
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpagorm"
)

func SQLRepo[T any](instanceName ...string) gpa.MigratableRepository[T] {
	return gpagorm.GetRepository[T](instanceName...)
}
