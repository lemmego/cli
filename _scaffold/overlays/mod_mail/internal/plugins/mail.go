package plugins

import (
	"time"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/mail"
	mailsmtp "github.com/lemmego/mail/smtp"
)

// Mail registers the mailer.
//
// With no MAIL_HOST the log transport stays in place, and that is the right
// default rather than a placeholder: outside production it writes the
// rendered body as well as the envelope, so a verification link is readable
// in the log during development without a relay to send it through. A
// transport that silently dropped the body instead reads as a broken
// feature.
func Mail() app.Provider {
	return &mail.Provider{
		Transport: mailTransport(),
		From: mail.Address{
			Name:  config.MustEnv("MAIL_FROM_NAME", config.MustEnv("APP_NAME", "Lemmego")),
			Email: config.MustEnv("MAIL_FROM_ADDRESS", "hello@example.com"),
		},
		// Where the bodies live. A template is looked up by base name, and a
		// locale subdirectory overrides the base one — so resources/mail/fr/
		// needs only the bodies actually translated.
		Templates: config.MustEnv("MAIL_TEMPLATES", "resources/mail"),
	}
}

// mailTransport returns nil without a host, which leaves the log transport.
//
// Nil rather than a zero-valued SMTP transport, because one configured with
// no host fails on every send at the point where somebody is waiting for a
// password reset.
func mailTransport() mail.Transport {
	host := config.MustEnv("MAIL_HOST", "")
	if host == "" {
		return nil
	}

	return mailsmtp.New(mailsmtp.Config{
		Host:     host,
		Port:     config.MustEnv("MAIL_PORT", 587),
		Username: config.MustEnv("MAIL_USERNAME", ""),
		Password: config.MustEnv("MAIL_PASSWORD", ""),
		// A relay that accepts the connection and then never answers would
		// otherwise hold the request open for as long as it liked.
		Timeout: 10 * time.Second,
	})
}
