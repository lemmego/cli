package plugins

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/auth"
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/oauth2"
)

// OAuth2 runs this application as an OAuth 2.0 authorization *server*: other
// applications send people here to sign in, and this issues the tokens.
//
// It is not a client. Signing in with GitHub or Google is auth/social, which
// the authentication provider already wires.
//
// The store is not set, so the provider opens one over this application's own
// connection — the same pool everything else uses, rather than a second one
// with its own idea of the schema.
func OAuth2() app.Provider {
	cfg := oauth2.DefaultConfig()

	// The iss claim of every token. A wrong issuer breaks every consumer and
	// is invisible until somebody integrates, so it comes from the one
	// setting that already has to be right.
	cfg.Issuer = config.MustEnv("OAUTH_ISSUER", config.MustEnv("APP_URL", "http://localhost:1490"))

	return &oauth2.Provider{
		Config: cfg,

		// Who the consent screen is asking. Returning false sends the visitor
		// to the login route instead, which is what should happen when an
		// authorization request arrives from somebody not signed in.
		//
		// GetID rather than the raw column, so the subject is a string the
		// store can hold whatever the model's key type is.
		ResourceOwner: func(c app.Context) (string, bool) {
			user, ok := auth.UserAs[*models.User](c)
			if !ok {
				return "", false
			}
			return user.GetID(), true
		},
	}
}
