package inputs

import (
	"github.com/lemmego/api/app"
)

type LoginInput struct {
	ctx      app.Context
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
}

func NewLoginInput(c app.Context) (*LoginInput, error) {
	input := &LoginInput{}
	err := c.ParseInput(input)
	if err != nil {
		return nil, err
	}
	input.ctx = c
	return input, input.Validate()
}

func (i *LoginInput) Validate() error {
	v := i.ctx.Validator()
	v.Field("email", i.Email).Email()
	v.Field("password", i.Password).Required()
	return v.Validate()
}
