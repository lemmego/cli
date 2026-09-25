package repos

import (
	"context"

	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/orm"
)

type UserRepository struct {
	db *orm.DB
}

func User(a app.App) *UserRepository {
	return &UserRepository{db: getDB(a)}
}

// Create inserts a user. The ORM runs the model's BeforeCreate hook itself, so
// the password is hashed without the repository having to remember to do it.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.Model[models.User]().Create(ctx, user)
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.db.Model[models.User]().Where(orm.Eq("email", email)).First(ctx)
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.db.Model[models.User]().Where(orm.Eq("email", email)).Exists(ctx)
}
