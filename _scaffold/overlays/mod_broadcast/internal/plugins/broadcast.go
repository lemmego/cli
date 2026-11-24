package plugins

import (
	"fmt"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/broadcast"
	redisbus "github.com/lemmego/broadcast/bus/redis"
	broadcastprovider "github.com/lemmego/broadcast/provider"
)

// Broadcast mounts the realtime endpoints: SSE and WebSocket, both.
//
// After auth, because a private channel is authorised against the signed-in
// user and the auth service has to be registered before a subscriber
// arrives.
//
// **AllowedOrigins is required rather than defaulted, and that is a security
// property rather than a configuration style.** A WebSocket is not subject
// to CORS: without an allowlist, any page on the internet could open an
// authenticated connection carrying the reader's cookies and receive
// everything on their channels. Set it to the origins this application is
// actually served from.
func Broadcast() app.Provider {
	p := &broadcastprovider.Provider{
		AllowedOrigins: []string{config.MustEnv("APP_URL", "http://localhost:1490")},
		BusFor:         broadcastBus(),

		Config: broadcast.Config{
			// Opted into, because a channel without the "private-" prefix is
			// refused by default — "an unauthenticated megaphone is a
			// decision, not a default". The demo channel below is public, so
			// this is what makes it reachable.
			//
			// Turn it off once every channel in the application is private,
			// which is the usual end state: private-user.<me> needs no
			// configuration at all and is what notify publishes to.
			AllowPublic: true,
		},

		// A fallback for an api that does not dispatch its own shutdown
		// event before http.Server.Shutdown. These connections are
		// long-lived by construction, so a provider that does not close them
		// hangs the process on every graceful stop. signal.Notify
		// multicasts, so this steals nothing from the application's own
		// handler.
		DrainOnSignal: true,
	}

	// A channel pattern is a rule about who may subscribe, and it is checked
	// on connect *and* again on every subscribe — a connection that can add
	// channels later must not be authorised once at the start.
	//
	// private-user.{subject} is registered by the provider itself, using the
	// signed-in user's own id as the subject, so a reader can only ever
	// subscribe to their own. This one is the public counterpart: no rule to
	// apply, because there is nothing to protect — and it needs AllowPublic
	// above, which is the knob worth knowing about before wondering why a
	// subscribe answers 403.
	//
	// A name that does not match any registered rule is refused the same way
	// as one that is not yours, deliberately: a difference between "no such
	// channel" and "not for you" is a channel-enumeration oracle.
	p.Channel("demo", func(c app.Context, params broadcastprovider.Params) error {
		return nil
	})

	return p
}

// broadcastBus returns memory for one process and Redis for more than one.
//
// An unreachable Redis fails the boot rather than falling back, because a
// fallback passes staging — where there is one replica and every subscriber
// is local — and then delivers half the events in production.
func broadcastBus() func(a app.App) (broadcast.Bus, error) {
	return func(a app.App) (broadcast.Bus, error) {
		switch config.MustEnv("BROADCAST_BUS", "memory") {
		case "memory":
			return broadcast.NewMemoryBus(), nil
		case "redis":
			return redisbus.New(redisbus.Config{
				Addr:     fmt.Sprintf("%s:%d", config.MustEnv("REDIS_HOST", "localhost"), config.MustEnv("REDIS_PORT", 6379)),
				Password: config.MustEnv("REDIS_PASSWORD", ""),
				// A database of its own, so flushing the cache does not
				// disconnect every subscriber.
				DB: config.MustEnv("BROADCAST_REDIS_DB", 2),
			})
		default:
			return nil, fmt.Errorf("BROADCAST_BUS must be memory or redis")
		}
	}
}
