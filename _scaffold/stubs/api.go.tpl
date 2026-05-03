package routes

import (
	"github.com/lemmego/api/app"
	{{- if .EnableAuth}}
	"fmt"
	"net/http"

	"github.com/lemmego/auth"
	"github.com/lemmego/gpa"
	"github.com/lemmego/lemmego/internal/inputs"
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/lemmego/internal/repos"
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
			app.Get[*auth.Auth](a).Logout(c.RequestContext())
			c.SetCookie(&http.Cookie{Name: "jwt", Value: "", Path: "/", MaxAge: -1})
			return c.JSON(app.M{"message": "Logged out successfully"})
		})

		apiGroup.Get("/me", auth.Protected, func(c app.Context) error {
			return app.M{"user": auth.AuthUser(c)}
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

			if err := repos.User().Create(c.RequestContext(), user); err != nil {
				return err
			}

			return app.M{"user": user, "message": "registration successful"}
		})

		apiGroup.Post("/login", func(c app.Context) error {
			input, err := inputs.NewLoginInput(c)
			if err != nil {
				return err
			}

			user, err := repos.User().QueryOne(c.RequestContext(), gpa.Where("email", "=", input.Email))
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
