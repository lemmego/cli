package routes

import (
	"fmt"
	"time"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/api/db"
	"github.com/lemmego/lemmego/internal/billing"
	"github.com/lemmego/pay"
)

// PayRoutes is the worked example: start a checkout, receive the webhook,
// read the entitlement back.
func PayRoutes(a app.App) {
	r := a.Router()
	r.Post("/pay/checkout", startCheckout)
	r.Get("/pay/entitlement/{reference}", readEntitlement)

	// The webhook. The module's own handler verifies the signature, rejects
	// a replay and hands the event to the store — so there is nothing for
	// this application to write here, and nothing it can get wrong.
	//
	// **It must be exempt from CSRF**, because Stripe cannot send a token
	// and has no session. The signature is what replaces that protection:
	// it is an HMAC over the timestamp *and* the body, so a forged or
	// replayed request fails it. Add "/pay/webhook" to the CSRF
	// middleware's exclusions in bootstrap/http_middleware.go, and keep the
	// reason next to the exemption.
	r.Post("/pay/webhook", pay.WebhookHandler())
}

func startCheckout(c app.Context) error {
	billingSvc, ok := pay.Get(c.App())
	if !ok {
		return fmt.Errorf("no billing: add plugins.Pay() to LoadProviders")
	}

	var body struct {
		PriceID   string `json:"price_id"`
		Email     string `json:"email"`
		Reference string `json:"reference"`
	}
	if err := c.DecodeJSON(&body); err != nil {
		return c.SetStatus(422).JSON(app.M{"error": "send price_id, email and reference"})
	}

	base := config.MustEnv("APP_URL", "http://localhost:1490")

	checkout, err := billingSvc.Checkout(c.RequestContext(), pay.CheckoutRequest{
		PriceID: body.PriceID,
		Email:   body.Email,

		// How the webhook finds out who this payment belongs to. It is the
		// only thing tying Stripe's customer back to a row in this
		// application, so it has to be set here — at the one point where
		// the person is known.
		ClientReference: body.Reference,

		SuccessURL: base + "/pay/success",
		CancelURL:  base + "/pay/cancelled",
	})
	if err != nil {
		return c.SetStatus(502).JSON(app.M{"error": err.Error()})
	}

	// The URL is where the person goes. Nothing is entitled yet, and the
	// success redirect will not change that either — only the webhook does.
	return c.JSON(app.M{"checkout_url": checkout.URL})
}

func readEntitlement(c app.Context) error {
	conn, ok := db.Resolve(c.App())
	if !ok {
		return fmt.Errorf("no database connection")
	}

	record, err := billing.New(conn).Active(c.RequestContext(), c.Param("reference"))
	if err != nil {
		return err
	}

	return c.JSON(app.M{
		// Read from the stored row, which the webhook wrote. A visitor who
		// opens the success URL directly, without paying, gets false here —
		// which is the whole reason entitlement is not read from the
		// redirect.
		"entitled":     record.Entitled(time.Now().UTC()),
		"subscription": record,
	})
}
