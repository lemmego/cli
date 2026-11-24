package plugins

import (
	"context"

	"github.com/lemmego/api/config"
	"github.com/lemmego/seo"
)

// staticPages is the sitemap source for the pages that are not rows.
//
// Every project has some: a landing page, an about page, terms. They are
// listed rather than discovered because nothing can discover them — a route
// table knows the paths but not which of them are worth indexing, and most
// are not.
//
// It is also the shape to copy for a real source. The contract pages rather
// than returning a slice, which is the whole point of it: a sitemap for a
// million rows must not become a million-element slice in memory, and the
// 50,000-URL limit per file means it never has to.
type staticSource struct {
	base string
	urls []seo.URL
}

func staticPages() seo.Source {
	base := config.MustEnv("APP_URL", "http://localhost:1490")

	return staticSource{
		base: base,
		urls: []seo.URL{
			{
				Loc:        base + "/",
				LastMod:    deployedAt,
				ChangeFreq: "weekly",
				// Relative to this site's own pages and nothing else.
				// Priority says which of your pages matters most, not how
				// your site compares to anybody else's, so one page at 1.0
				// and the rest below it is the only arrangement that says
				// anything.
				Priority: 1.0,
			},
		},
	}
}

// Name appears in the shard URLs: "pages" gives /sitemap-pages-1.xml.
func (s staticSource) Name() string { return "pages" }

func (s staticSource) Count(ctx context.Context) (int, error) { return len(s.urls), nil }

func (s staticSource) Page(ctx context.Context, offset, limit int) ([]seo.URL, error) {
	if offset >= len(s.urls) {
		return nil, nil
	}
	end := offset + limit
	if end > len(s.urls) {
		end = len(s.urls)
	}
	return s.urls[offset:end], nil
}
