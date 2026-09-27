package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/uptrace/bun"

	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
	"github.com/lemmego/lemmego/internal/models"
)

type UserRepository struct {
	db *bun.DB
}

func User(a app.App) *UserRepository {
	return &UserRepository{db: getDB(a)}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := user.BeforeCreate(ctx); err != nil {
		return err
	}
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
	user := new(models.User)
	if err := r.db.NewSelect().Model(user).Where("id = ?", pk).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id %d", auth.ErrUserNotFound, pk)
		}
		return nil, err
	}
	return user, nil
}
