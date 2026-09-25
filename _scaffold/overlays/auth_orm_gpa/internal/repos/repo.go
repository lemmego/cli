package repos

import (
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpaorm"
)

func SQLRepo[T any](instanceName ...string) gpa.MigratableRepository[T] {
	provider := gpa.MustGet[*gpaorm.Provider](instanceName...)
	return provider.Repository[T]()
}
