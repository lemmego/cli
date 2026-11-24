package routes

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/api/config"
	"github.com/lemmego/seo"
)

// SeoRoutes is the worked example: one page whose head is built in Go.
//
// /sitemap.xml, its shards and /robots.txt are not here — the seo provider
// mounts those itself from internal/plugins/seo.go.
func SeoRoutes(a app.App) {
	a.Router().Get("/seo/preview", seoPreview)
}

// seoPreview renders the <head> seo would emit for a page, as text.
//
// Returning it as text rather than rendering a page is what makes this
// example work in every preset: a project on Inertia, templ or Go templates
// renders differently, and the thing being demonstrated is the same in all
// three.
//
// **The head is built in Go, and that is the point rather than an
// implementation detail.** Rendering it in the page component instead puts
// every meta tag behind JavaScript, which means a server-rendering failure —
// the kind that answers 200 and looks perfect to a human — serves a page with
// no title, no canonical URL and no structured data to a crawler. Built here,
// it survives that.
//
// In a real handler the Meta goes to the renderer instead:
//
//	seo.Share(c, meta)        // Inertia reads it from the context
//	return c.HTML(page)       // or a template that calls {{ .seo.Render }}
func seoPreview(c app.Context) error {
	meta := seo.Meta{
		Title:       "A worked example",
		Description: "One page, with its title, canonical URL and structured data built in Go.",

		// Absolute and self-referencing. A canonical URL is how two paths
		// that serve one page — with and without a trailing slash, with and
		// without a tracking parameter — are told to count as one, and a
		// relative one is ignored by every crawler.
		//
		// Written out here because this example renders the Meta directly.
		// Through Share it may be a bare path: the site defaults prepend
		// BaseURL, which is the usual way to write it.
		Canonical: config.MustEnv("APP_URL", "http://localhost:1490") + c.Request().URL.Path,

		Type: "article",

		// Structured data is a machine-readable claim, so only make the ones
		// that are true. An Organization block on a page about a company
		// somebody else added, for instance, is a claim this application
		// cannot make.
		JSONLD: []any{
			map[string]any{
				"@context":    "https://schema.org",
				"@type":       "Article",
				"headline":    "A worked example",
				"description": "One page, with its head built in Go.",
			},
		},
	}

	// Share puts the rendered head on the context for a renderer to read,
	// and is the call a real page makes immediately before responding. It
	// also applies the site-wide defaults — the site name, the robots
	// directive, og:locale from the reader's language, and the fallback card
	// image — so a page that sets only a title still emits a complete head.
	//
	// Called here so the example exercises the same path a real page does.
	seo.Share(c, meta)

	// Render alone, which is what this endpoint prints, is the page's own
	// tags *without* those defaults. That is why the output below carries no
	// og:site_name and no robots directive: nothing is missing from the
	// Meta, the defaults are simply applied on the other path.
	return c.Text([]byte(meta.Render()))
}
