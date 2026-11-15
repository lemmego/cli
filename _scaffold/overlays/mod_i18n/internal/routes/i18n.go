package routes

import (
	"github.com/lemmego/api/app"
	i18nprovider "github.com/lemmego/i18n/provider"
)

// I18nRoutes is the worked example: one endpoint that answers in the
// reader's language.
//
// Try it: /i18n/demo, then /i18n/demo?lang=fr, then the same with
// `Accept-Language: fr`. The third one needs no query parameter and no
// cookie, which is the path a real visitor arrives on.
func I18nRoutes(a app.App) {
	a.Router().Get("/i18n/demo", i18nDemo)
}

func i18nDemo(c app.Context) error {
	// c.T is the short form and reads the request's translator off the
	// context, which is what a handler normally wants. From is here instead
	// because this example also reports the resolved locale, and that is on
	// the translator rather than on the context.
	t := i18nprovider.From(c)

	return c.JSON(app.M{
		// The locale the negotiation settled on. It is resolved lazily and
		// freezes on the first translation of the request — deliberately: a
		// translator that kept answering the current value would render the
		// top of a page in one language and the bottom in another, which is
		// harder to notice than a whole page in the wrong one.
		"locale": t.Locale(),

		"greeting": t.Get("messages.greeting", map[string]any{"name": "Ada"}),

		// A plural, selected by CLDR rules rather than by n == 1. French
		// uses the "one" form for both 0 and 1, which is the case a
		// hand-rolled check gets wrong — and :count is bound without being
		// passed.
		"zero": t.Choice("messages.items", 0),
		"one":  t.Choice("messages.items", 1),
		"many": t.Choice("messages.items", 7),

		// Untranslated in French on purpose, so the fallback to the default
		// locale is visible here rather than only in a test.
		"nav_home": t.Get("messages.nav.home"),
	})
}
