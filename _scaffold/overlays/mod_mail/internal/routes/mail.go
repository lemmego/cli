package routes

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/lemmego/internal/mailables"
	"github.com/lemmego/mail"
)

// MailRoutes is the worked example: one endpoint that sends one email.
func MailRoutes(a app.App) {
	a.Router().Post("/mail/welcome", sendWelcome)
}

// sendWelcome builds a Mailable and hands it to the mailer.
//
// Send rather than Queue, so this example needs no worker running and the
// result — or the failure — is visible in the response. Real sending wants
// the other one:
//
//	mail.Queue(c.RequestContext(), msg)
//
// Queue renders the body eagerly on this goroutine and enqueues a message
// that carries everything it needs, so a dead relay becomes a retried job
// rather than a failed request, and nothing in the worker needs the
// templates. That eager render is also why anything per-request — the
// reader's locale, their display name — has to be resolved before the
// enqueue and not inside the job.
func sendWelcome(c app.Context) error {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.DecodeJSON(&body); err != nil {
		return c.SetStatus(422).JSON(app.M{"error": "send a JSON body with name and email"})
	}
	if body.Email == "" {
		return c.SetStatus(422).JSON(app.M{"error": "email is required"})
	}

	msg := mailables.Welcome{
		Name:  body.Name,
		Email: body.Email,
		// Built here because this is the only layer that knows the
		// application's own base URL. A Mailable that read it from the
		// environment itself could not be asserted on in a test without
		// setting one.
		URL: config.MustEnv("APP_URL", "http://localhost:8080"),
	}

	if err := mail.Send(c.RequestContext(), msg); err != nil {
		return c.SetStatus(502).JSON(app.M{"error": err.Error()})
	}

	return c.JSON(app.M{"sent": body.Email})
}
