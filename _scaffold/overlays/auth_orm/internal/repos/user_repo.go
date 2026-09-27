package repos

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
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
	user, err := r.db.Model[models.User]().Where(orm.Eq("id", pk)).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return nil, fmt.Errorf("%w: id %d", auth.ErrUserNotFound, pk)
	}
	return user, err
}
