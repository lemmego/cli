package inputs

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/db"
)

type RegisterInput struct {
	ctx                  app.Context
	Name                 string `json:"name" form:"name"`
	Email                string `json:"email" form:"email"`
	Password             string `json:"password" form:"password"`
	PasswordConfirmation string `json:"password_confirmation" form:"password_confirmation"`
}

func NewRegisterInput(c app.Context) (*RegisterInput, error) {
	input := &RegisterInput{}
	err := c.ParseInput(input)
	if err != nil {
		return nil, err
	}
	input.ctx = c
	return input, input.Validate()
}

func (i *RegisterInput) Validate() error {
	v := i.ctx.Validator()
	v.Field("name", i.Name).Required().Max(255)
	// Check, not Custom. Custom answers (bool, string) and so has no way to
	// say "I could not tell": a uniqueness lookup that failed because the
	// database was unreachable came back as false, and the person signing up
	// was told their email address was already in use. Check takes a rule
	// that can also return an error, which v.Err reports separately below.
	v.Field("email", i.Email).Required().Email().
		Check(db.Unique(i.ctx.App(), "users", "email"))
	v.Field("password", i.Password).Required().Min(8)
	v.Field("password_confirmation", i.PasswordConfirmation).Required().Equals(i.Password)
	// Err is a rule that could not run, as opposed to a value somebody can
	// fix. Returning it separately lets the caller answer 503 instead of
	// blaming the input for an outage.
	if err := v.Err(); err != nil {
		return err
	}
	return v.Validate()
}
