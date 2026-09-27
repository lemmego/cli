package repos

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"gorm.io/gorm"

	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
	"github.com/lemmego/lemmego/internal/models"
)

type UserRepository struct {
	db *gorm.DB
}

func User(a app.App) *UserRepository {
	return &UserRepository{db: getDB(a)}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := user.BeforeCreate(ctx); err != nil {
		return err
	}
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

// FindByID resolves the identifier a credential carries back to a row.
//
// auth works in string ids — a JWT subject is a string, and UserProvider.GetID
// returns one — and this is the single place that knows the column behind it
// is a uint64.
//
// An id that does not parse is a user that does not exist, not a failure. A
// credential issued before the identifier format changed, or a fabricated one,
// lands here; reporting it as an error would make auth answer 503, so on the
// day of an upgrade a wave of stale tokens would read as an outage instead of
// asking one person to sign in again.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	pk, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is not a user id", auth.ErrUserNotFound, id)
	}
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, pk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id %d", auth.ErrUserNotFound, pk)
		}
		return nil, err
	}
	return &user, nil
}
