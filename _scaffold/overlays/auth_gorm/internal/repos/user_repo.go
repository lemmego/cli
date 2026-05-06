package repos

import (
	"context"

	"gorm.io/gorm"

	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func User(a app.App) *UserRepository {
	return &UserRepository{db: getDB(a)}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}
