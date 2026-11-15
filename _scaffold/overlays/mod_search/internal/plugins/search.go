package plugins

import (
	"fmt"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/db"
	"github.com/lemmego/lemmego/internal/searchable"
	"github.com/lemmego/search"
	searchprovider "github.com/lemmego/search/provider"

	// Registers the engines that are not built in — memory and Meilisearch —
	// so SEARCH_ENGINE can name one. The postgres engine is not here: it
	// lives in the module's root and registers itself, because it needs
	// nothing but database/sql and so costs a consumer nothing to link.
	_ "github.com/lemmego/search/engines"
)

// SearchProvider is the module's provider with this application's
// importables filled in.
//
// A wrapper rather than a literal, because Importables hold a connection and
// the provider list is written before the container exists. Embedding the
// module's provider rather than reimplementing it means its commands —
// search:import and search:flush — and its publishable config are promoted
// and keep working.
type SearchProvider struct {
	*searchprovider.Provider
}

// Search registers the finder and the import and flush commands.
//
// After the database connector, whose pool the postgres engine searches.
//
// A configured engine that cannot be reached **fails the boot**, and there
// is deliberately no lenient mode: a cache that keeps nothing is slow but
// correct, whereas a search that finds nothing looks exactly like an empty
// database and is the kind of outage somebody notices a week later.
func Search() app.Provider {
	return &SearchProvider{
		Provider: &searchprovider.Provider{
			// Which engine to use comes from internal/configs/search.go, so
			// a deployment changes engine with an environment variable
			// rather than by editing Go. The pool cannot: no configuration
			// string can name a *sql.DB, which is why this one is a
			// function.
			DB: func(a app.App) (search.Querier, error) {
				conn, ok := db.Resolve(a)
				if !ok {
					return nil, fmt.Errorf("search: no database connection")
				}
				return conn.SQLDB(), nil
			},
		},
	}
}

// Provide fills in the importables and then boots the module's provider.
func (p *SearchProvider) Provide(a app.App) error {
	// What `lemmego run search:import` walks, one per index.
	//
	// Listed rather than discovered. Go has no equivalent of Laravel's model
	// discovery, and a reflective registry that silently missed a model
	// would be worse than a list — the symptom is one collection quietly
	// absent from search, which nothing fails on.
	//
	// Meaningless to the postgres engine, which has no index of its own to
	// fill, and written anyway: switching to Meilisearch is then a
	// configuration change plus one import run, rather than writing a walker
	// under time pressure.
	if conn, ok := db.Resolve(a); ok {
		p.Importables = []search.Importable{
			searchable.NotesImporter{DB: conn.SQLDB()},
		}
	}

	return p.Provider.Provide(a)
}
