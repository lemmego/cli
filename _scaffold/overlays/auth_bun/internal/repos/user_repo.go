package repos

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/models"
)

type UserRepository struct {
	db *bun.DB
}

func User(a app.App) *UserRepository {
	return &UserRepository{db: getDB(a)}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.NewInsert().Model(user).Exec(ctx)
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	user := new(models.User)
	err := r.db.NewSelect().Model(user).Where("email = ?", email).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.db.NewSelect().Model((*models.User)(nil)).Where("email = ?", email).Exists(ctx)
}
