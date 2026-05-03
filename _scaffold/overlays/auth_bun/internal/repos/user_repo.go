package repos

import (
	"github.com/lemmego/gpa"
	"github.com/lemmego/lemmego/internal/models"
)

type UserRepository struct {
	gpa.MigratableRepository[models.User]
}

func User(instanceName ...string) *UserRepository {
	repo := SQLRepo[models.User](instanceName...)
	return &UserRepository{repo}
}
