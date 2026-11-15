package plugins

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/api/db"
	"github.com/lemmego/lemmego/internal/billing"
	"github.com/lemmego/pay"
	paystripe "github.com/lemmego/pay/stripe"
)

// Pay registers one hosted checkout and one webhook.
//
// Thin on purpose: the expensive mistakes in billing are not in the API
// calls, they are in what the application believes afterwards. See
// internal/billing, which is the half this module cannot write for you.
func Pay() app.Provider {
	return &pay.Provider{
		Gateway: paystripe.New(paystripe.Config{
			SecretKey: config.MustEnv("STRIPE_SECRET_KEY", ""),
		}),

		// A function, because the provider list is built before the
		// container exists and the store needs a connection.
		Store: func(a app.App) pay.Store {
			conn, ok := db.Resolve(a)
			if !ok {
				// Nil rather than a panic: the provider reports a missing
				// store with its own message, which names the field.
				return nil
			}
			return billing.New(conn)
		},

		// Two secrets, so one can be rotated without an interval in which
		// events are refused — and a refused event is money moving with
		// nothing recording it.
		Secrets: []string{
			config.MustEnv("STRIPE_WEBHOOK_SECRET", ""),
			config.MustEnv("STRIPE_WEBHOOK_SECRET_PREVIOUS", ""),
		},

		// Livemode is deliberately unset: it is derived from the secret
		// key's own prefix, so a flag here cannot disagree with the key
		// actually in use.
	}
}
