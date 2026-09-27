package routes

import (
	"github.com/lemmego/api/app"
	{{- if .EnableAuth}}
	"fmt"

	"github.com/lemmego/auth"
	"github.com/lemmego/lemmego/internal/inputs"
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/lemmego/internal/repos"
	{{- if .EnableGPA}}
	"github.com/lemmego/gpa"
	{{- end}}
	{{- end}}
)

func ApiRoutes(a app.App) {
	r := a.Router()
	apiGroup := r.Group("/api")
	{
		apiGroup.Get("/ping", func(c app.Context) error {
			return app.M{"message": "pong"}
		})
		{{- if .EnableAuth}}

		apiGroup.Post("/logout", func(c app.Context) error {
			auth.Logout(c)
			return c.JSON(app.M{"message": "Logged out successfully"})
		})

		// UserAs is the typed accessor: the loader configured in
		// bootstrap/providers.go put a *models.User on the context, so this
		// reads back the application's own type rather than whatever shape
		// the credential happened to have. It is false for a token that
		// authenticates a machine and no user.
		apiGroup.Get("/me", auth.Protected, func(c app.Context) error {
			user, ok := auth.UserAs[*models.User](c)
			if !ok {
				return c.Error(403, fmt.Errorf("this endpoint needs a user, not a client credential"))
			}
			return app.M{"user": user}
		})

		apiGroup.Post("/register", func(c app.Context) error {
			input, err := inputs.NewRegisterInput(c)
			if err != nil {
				return err
			}

			user := &models.User{
				Name:     input.Name,
				Email:    input.Email,
				Password: input.Password,
			}

			{{- if .EnableGPA}}
			if err := repos.User().Create(c.RequestContext(), user); err != nil {
			{{- else}}
			if err := repos.User(a).Create(c.RequestContext(), user); err != nil {
			{{- end}}
				return err
			}

			return app.M{"user": user, "message": "registration successful"}
		})

		apiGroup.Post("/login", func(c app.Context) error {
			input, err := inputs.NewLoginInput(c)
			if err != nil {
				return err
			}

			{{- if .EnableGPA}}
			user, err := repos.User().QueryOne(c.RequestContext(), gpa.Where("email", "=", input.Email))
			{{- else}}
			user, err := repos.User(a).FindByEmail(c.RequestContext(), input.Email)
			{{- end}}
			if err != nil {
				return err
			}

			result := auth.Login(c, user, input.Email, input.Password)

			if result.Err != nil {
				return c.Error(422, fmt.Errorf("login failed: %w", result.Err))
			}

			return app.M{"message": "login successful", "token": result.JwtToken, "user": user}
		})
		{{- end}}
	}
}
