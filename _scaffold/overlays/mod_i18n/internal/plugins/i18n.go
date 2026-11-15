package plugins

import (
	"github.com/lemmego/api/app"
	i18nprovider "github.com/lemmego/i18n/provider"
)

// I18n registers the translator and resolves each request's locale.
//
// Two locales rather than one, and that is deliberate: a supported list of
// one proves nothing. Every negotiation path, every fallback and the whole
// shape of what reaches the browser is untested until a second locale
// exists — and resources/lang/fr/ is only partly translated here, so the
// fallback is exercised by a real page rather than by a fixture.
//
// The locale is resolved from, in order: an explicit Prefer call, ?lang=,
// the signed-in reader's stored preference, a cookie, and Accept-Language.
// A cookie and not the session, because an application may legitimately run
// with no session store and a session-backed preference would then silently
// do nothing.
func I18n() app.Provider {
	return &i18nprovider.Provider{
		// The first is the default and the fallback.
		Supported: []string{"en", "fr"},

		// Enabled here although the module leaves it off by default, because
		// a query parameter is a switch anybody can flip on anybody else's
		// link — fine for a language picker, wrong as a framework default.
		//
		// The middleware persists it to the cookie when it resolves, which
		// it has to: validation errors are rendered in the locale of the
		// failing POST and read back after a redirect, so without that a
		// reader who arrived on ?lang=fr would see French everywhere except
		// their own error messages.
		QueryKey: "lang",

		// A module ships its own messages under a namespace, and mounting is
		// what makes them resolvable. authz, for one, keys five strings a
		// reader can actually see:
		//
		//	Mount: map[string]fs.FS{"authz": authz.Catalogue()},
		//
		// which makes authz::errors.forbidden resolvable. An override goes
		// in resources/lang/vendor/authz/<locale>/ and
		// `lemmego run publish --tags=authz-i18n` writes the English there
		// to start from.

		// UserLocale is deliberately unset. There is no locale column on
		// anything yet, and a cookie serves every reader who has visited
		// since choosing. The column earns its migration the day somebody
		// has to be emailed in a language they chose on another device.
	}
}
