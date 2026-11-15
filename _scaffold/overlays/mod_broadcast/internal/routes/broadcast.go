package routes

import (
	"fmt"

	"github.com/lemmego/api/app"
	"github.com/lemmego/broadcast"
	broadcastprovider "github.com/lemmego/broadcast/provider"
)

// BroadcastRoutes is the worked example: one endpoint that publishes.
//
// The subscriber endpoints are not here — the provider mounts those itself at
// /broadcast/sse and /broadcast/ws. Open one of these in a browser console
// and then POST to /broadcast/publish:
//
//	const es = new EventSource("/broadcast/sse?channel=demo")
//	es.addEventListener("event", (e) => console.log(JSON.parse(e.data)))
//
// The parameter is `channel`, singular and repeatable — one connection can
// carry several, which is the point of a multiplexed stream: a browser is
// capped at about six connections per origin over HTTP/1.1, and an event
// stream holds one open for as long as the tab is.
func BroadcastRoutes(a app.App) {
	a.Router().Post("/broadcast/publish", publishDemo)
}

func publishDemo(c app.Context) error {
	caster, ok := broadcastprovider.Get(c.App())
	if !ok {
		return fmt.Errorf("no broadcaster: add plugins.Broadcast() to LoadProviders")
	}

	event, err := broadcast.NewEvent("demo.message", app.M{
		"text": "hello from the server",
	})
	if err != nil {
		return err
	}

	// **Publish after the transaction commits, never inside it.** Inside,
	// the event reaches a browser before the row is visible, so the client
	// refetches and finds nothing — and if the transaction then rolls back,
	// somebody has been told about a thing that does not exist.
	//
	// A nil error means "accepted for delivery" and never "delivered". A
	// client too slow to keep up is disconnected rather than waited for,
	// which is the right trade for a bounded buffer: an unbounded one is a
	// memory leak one slow client wide, and a client that silently misses an
	// event is worse than one that reconnects.
	if err := caster.Publish(c.RequestContext(), "demo", event); err != nil {
		// Worth distinguishing in a real handler: an error wrapping
		// ErrPartialPublish means local subscribers were served and remote
		// ones were not. It is not retried for you — a queued job can retry,
		// an HTTP handler mostly cannot.
		return c.SetStatus(502).JSON(app.M{"error": err.Error()})
	}

	return c.JSON(app.M{"published": "demo"})
}
