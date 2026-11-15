package routes

import (
	"time"

	"github.com/lemmego/api/app"
	"github.com/lemmego/ratelimit"
)

// RatelimitRoutes is the worked example: one route behind a limiter.
//
// There is no provider to register. The counter lives in the cache, which
// the application already has, and the limiter is route middleware — so
// where it applies is a routing decision and is written where the routes
// are.
func RatelimitRoutes(a app.App) {
	r := a.Router()

	// PerIP, because this route is reachable by somebody who is not signed
	// in. Use it for anything an anonymous visitor can hit: sign-in,
	// registration, password reset, a contact form.
	//
	// The key includes the route, so a limit on sign-in does not also spend
	// somebody's registration attempts.
	r.Get("/ratelimit/demo", ratelimit.Middleware(ratelimit.PerIP(5, time.Minute)), func(c app.Context) error {
		return c.JSON(app.M{"ok": true})
	})

	// PerUser is what an action only a signed-in person can take wants —
	// posting, voting, commenting — and the reason is not symmetry: an IP is
	// shared by an office, a university and a mobile carrier's NAT, so
	// limiting those as one caller punishes the wrong people. It falls back
	// to the IP for an anonymous request, so it is safe on a mixed route.
	//
	//	r.Post("/comments", ratelimit.Middleware(ratelimit.PerUser(10, time.Minute)), …)
}
