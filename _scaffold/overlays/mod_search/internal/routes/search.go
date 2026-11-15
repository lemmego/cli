package routes

import (
	"fmt"
	"strconv"

	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/searchable"
	"github.com/lemmego/search"
)

// SearchRoutes is the worked example: one search box over one table.
func SearchRoutes(a app.App) {
	a.Router().Get("/search", runSearch)
}

func runSearch(c app.Context) error {
	finder, ok := app.Lookup[*search.Finder](c.App())
	if !ok {
		return fmt.Errorf("no finder: add plugins.Search() to LoadProviders")
	}

	offset, _ := strconv.Atoi(c.Query("offset"))

	results, err := finder.Search(c.RequestContext(), search.Query{
		// Whatever somebody typed, taken raw. It goes through
		// websearch_to_tsquery, which is the only one of Postgres's four
		// conversion functions that cannot raise a syntax error on arbitrary
		// input — so `a & | b` in a search box is a result set rather than a
		// 500.
		Text: c.Query("q"),

		// Clamped by the module, once, so a hand-written limit in a query
		// string cannot ask for the whole table.
		Limit:  20,
		Offset: offset,

		// Who is asking. Bound as a parameter wherever a visibility rule
		// writes the placeholder, never interpolated. Left nil here because
		// this example's rule does not mention the viewer — and note that
		// Postgres refuses a statement that binds a parameter it never
		// references, which is why the module only binds it when the rule
		// uses it.
		Viewer: nil,
	}, searchable.Notes())
	if err != nil {
		return c.SetStatus(422).JSON(app.M{"error": err.Error()})
	}

	// One group per source, in the order they were passed, including the
	// empty ones — and never blended into a single ranked list. ts_rank is
	// not comparable across documents of different shapes, so merging two
	// sources produces an ordering that looks authoritative and means
	// nothing.
	//
	// Empty is its own answer: a page needs to tell "type something" apart
	// from "no results", and an empty query is answered without running a
	// single index scan.
	return c.JSON(app.M{
		"query":  results.Text,
		"empty":  results.Empty,
		"total":  results.Total,
		"groups": results.Groups,
	})
}
