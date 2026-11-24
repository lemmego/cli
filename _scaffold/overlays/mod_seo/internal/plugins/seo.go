package plugins

import (
	"time"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/seo"
)

// deployedAt stamps the static pages in the sitemap.
//
// Taken once at process start rather than per request, so every page in one
// deployment reports the same lastmod and the value does not creep forward
// while a crawler is part-way through reading the file.
var deployedAt = time.Now().UTC()

// Seo registers the meta, JSON-LD and sitemap helpers.
//
// NoIndex everywhere but production, which is the setting worth having by
// default: a staging deployment indexed alongside the real one splits its
// ranking, and nobody notices for months because both pages look fine.
func Seo() app.Provider {
	return &seo.Provider{
		Defaults: seo.Defaults{
			BaseURL:  config.MustEnv("APP_URL", "http://localhost:1490"),
			SiteName: config.MustEnv("APP_NAME", "Lemmego"),
			NoIndex:  config.MustEnv("APP_ENV", "development") != "production",

			// Open Graph asks for language_TERRITORY and silently ignores a
			// hyphen, so this is underscored. It is the site-wide fallback;
			// a page in another language sets its own through seo.Share.
			Locale: "en_US",
		},
		// Set, because the provider serves /sitemap.xml, its shards and
		// /robots.txt only when it is — a nil Sitemap registers the
		// defaults and no routes, which looks like a working install with
		// no sitemap.
		Sitemap: &seo.Sitemap{
			// Zero in development, so an edit shows up on the next request
			// rather than in fifteen minutes.
			CacheTTL: 0,
			Robots: seo.Robots{
				DisallowAll: config.MustEnv("APP_ENV", "development") != "production",
			},
		},
		// One source, as the shape to copy. A source yields the URLs of one
		// kind of page, and the sitemap handler pages through every source
		// it is given — so adding a model to the sitemap is adding a source
		// here rather than editing a template.
		//
		// A function rather than a value because a real source needs a
		// repository, and a repository needs a connection: LoadProviders
		// runs before anything is registered.
		Sources: func(a app.App) []seo.Source {
			return []seo.Source{staticPages()}
		},
	}
}
