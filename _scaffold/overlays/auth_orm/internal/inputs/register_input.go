package inputs

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/repos"
)

type RegisterInput struct {
	ctx                  app.Context
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
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
	v.Field("email", i.Email).Required().Email().Custom(func(v interface{}) (bool, string) {
		exists, err := repos.User(i.ctx.App()).ExistsByEmail(i.ctx.RequestContext(), i.Email)
		if err != nil {
			return false, "Error checking email uniqueness"
		}
		if exists {
			return false, "Email already in use"
		}
		return true, ""
	})
	v.Field("password", i.Password).Required().Min(8)
	v.Field("password_confirmation", i.PasswordConfirmation).Required().Equals(i.Password)
	return v.Validate()
}
