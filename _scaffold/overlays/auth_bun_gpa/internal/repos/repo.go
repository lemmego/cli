package repos

import (
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpabun"
)

func SQLRepo[T any](instanceName ...string) gpa.SQLRepository[T] {
	provider := gpa.MustGet[*gpabun.Provider](instanceName...)
	return provider.Repository[T]()
}
