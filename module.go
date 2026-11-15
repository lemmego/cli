package cli

import (
	"fmt"
	"sort"
	"strings"
)

// Module is an optional lemmego module a new project can be scaffolded with.
//
// These are the application-level modules — the ones that do something a
// product needs rather than something the framework needs. The
// infrastructure modules (api, the SQL connectors, cache, queue, session,
// inertia, log) are not on this list because they are not a choice in the
// same sense: a project either has a database or it does not, and the
// questions that decide those are asked separately.
//
// A string rather than a bool per module, for two reasons. The set is open —
// the framework has gained six of these in the last six phases and will gain
// more — and a []Module can be read off a --modules flag, sorted and
// compared, which eleven bools cannot without eleven lines of plumbing each
// time.
type Module string

const (
	ModuleAuthz     Module = "authz"
	ModuleBroadcast Module = "broadcast"
	ModuleI18n      Module = "i18n"
	ModuleMail      Module = "mail"
	ModuleMedia     Module = "media"
	ModuleNotify    Module = "notify"
	ModuleObserve   Module = "observe"
	ModulePay       Module = "pay"
	ModuleRatelimit Module = "ratelimit"
	ModuleSearch    Module = "search"
	ModuleSeo       Module = "seo"
)

// moduleSpec is what the CLI knows about one module.
//
// requires is checked by validate and used to withdraw an option from the
// interactive form, so a combination the project cannot boot is never
// offered rather than being offered and then refused — the rule the queue
// driver already follows.
type moduleSpec struct {
	name Module

	// summary is one line, shown in the picker.
	summary string

	// requires reports whether the rest of the configuration can support this
	// module, and returns the reason it cannot.
	//
	// A function rather than a list of flags, because the conditions are not
	// all of one kind: authz needs authentication, ratelimit needs a cache,
	// and three of them need a database for a table they create.
	requires func(c ProjectConfig) error
}

// AllModules is every module, in the order the picker shows them.
//
// Alphabetical, deliberately. Any other order encodes an opinion about which
// of these somebody should want, and the honest answer is that it depends on
// what they are building.
var AllModules = []moduleSpec{
	{
		name:    ModuleAuthz,
		summary: "Policies, ownership and a route gate",
		requires: func(c ProjectConfig) error {
			// A policy decides what this user may do to this record, so there
			// has to be a user. Without authentication every policy would be
			// asked about nobody and would have to answer no, which is a
			// working module that refuses everything.
			if !c.EnableAuth {
				return fmt.Errorf("authz needs authentication: a policy compares a record's owner against the signed-in user")
			}
			return nil
		},
	},
	{
		name:    ModuleBroadcast,
		summary: "Realtime events over SSE and WebSocket",
	},
	{
		name:    ModuleI18n,
		summary: "Translation, CLDR plurals and locale negotiation",
	},
	{
		name:    ModuleMail,
		summary: "SMTP, html/template bodies, queued sending",
	},
	{
		name:    ModuleMedia,
		summary: "Image ingest: orientation, resize, metadata stripped",
	},
	{
		name:    ModuleNotify,
		summary: "Notifications over database, mail and broadcast",
		requires: func(c ProjectConfig) error {
			// The database channel is the only one of the three that is
			// useful on its own — mail without an SMTP host sends nowhere a
			// person will look, and broadcast reaches only a browser that is
			// already open. So the in-app row is what the example uses, and
			// it needs a table.
			if !c.HasDatabase() {
				return fmt.Errorf("notify needs a database: its in-app channel stores a notifications row")
			}
			// Not a scaffold limitation. notify's SQL store ships statements
			// for Postgres and SQLite and refuses anything else by design:
			// "an untested dialect string is worse than an honest refusal,
			// because it looks like support right up to the point where
			// somebody's JSON column or partial index fails inside their
			// migration." Offering it on MySQL here would scaffold a project
			// whose notifications table cannot be created.
			if c.SQLDriver == SQLMySQL {
				return fmt.Errorf("notify has no MySQL store yet: its schema ships for postgres and sqlite only")
			}
			return nil
		},
	},
	{
		name:    ModuleObserve,
		summary: "Request-scoped debugging: what one request did, with timings",
	},
	{
		name:    ModulePay,
		summary: "One hosted checkout and one webhook, over Stripe",
		requires: func(c ProjectConfig) error {
			// Entitlement comes from the webhook and has to outlive the
			// process that received it, so there is nowhere to put it without
			// a database. A pay integration that forgets who paid on restart
			// is worse than none.
			if !c.HasDatabase() {
				return fmt.Errorf("pay needs a database: a subscription is written by the webhook and read on every request")
			}
			// The store's two writes are both upserts whose conflict target
			// is named — the idempotency insert and the out-of-order guard —
			// and ON CONFLICT is Postgres and SQLite syntax. MySQL wants
			// INSERT ... ON DUPLICATE KEY UPDATE, which cannot express the
			// WHERE that makes the older event lose. Shipping it with a note
			// saying so would be shipping a billing integration that
			// silently applies events in arrival order.
			if c.SQLDriver == SQLMySQL {
				return fmt.Errorf("pay has no MySQL store yet: its upserts need ON CONFLICT, which mysql does not have")
			}
			return nil
		},
	},
	{
		name:    ModuleRatelimit,
		summary: "Middleware and Allow, over the cache",
		requires: func(c ProjectConfig) error {
			// The counter lives in the cache. There is no in-process
			// fallback on purpose: one would work in development and then
			// count per replica in production, which is a limit of N times
			// what was asked for.
			if !c.HasCache() {
				return fmt.Errorf("ratelimit needs a cache: the window counter lives in it")
			}
			return nil
		},
	},
	{
		name:    ModuleSearch,
		summary: "Full-text search over the rows you already have",
		requires: func(c ProjectConfig) error {
			if !c.HasDatabase() {
				return fmt.Errorf("search needs a database: it searches this application's own rows")
			}
			// The default engine searches tsvector columns, which only
			// Postgres has. The alternatives are a Meilisearch instance,
			// which is infrastructure a new project should not be made to
			// install to have a search box, and the memory engine, which
			// forgets everything on restart. So this is offered where it
			// works out of the box and withheld where the honest answer is
			// "install something first".
			if c.SQLDriver != SQLPostgres {
				return fmt.Errorf("search needs postgres: its default engine searches tsvector columns, which mysql and sqlite do not have")
			}
			return nil
		},
	},
	{
		name:    ModuleSeo,
		summary: "Meta tags, JSON-LD, sitemaps and slugs",
	},
}

// moduleSpecs indexes AllModules by name.
var moduleSpecs = func() map[Module]moduleSpec {
	m := make(map[Module]moduleSpec, len(AllModules))
	for _, spec := range AllModules {
		m[spec.name] = spec
	}
	return m
}()

// KnownModule reports whether a name is a module this CLI can scaffold.
func KnownModule(name string) bool {
	_, ok := moduleSpecs[Module(name)]
	return ok
}

// ModuleNames lists every module name, for a flag's help text and an error.
func ModuleNames() []string {
	names := make([]string, 0, len(AllModules))
	for _, spec := range AllModules {
		names = append(names, string(spec.name))
	}
	return names
}

// ParseModules reads a --modules value: a comma-separated list, or "all".
//
// "all" means every module whose prerequisites this configuration meets,
// rather than every module that exists. The alternative is that
// `--modules all` fails on a project with no database, and the useful
// reading of "all" is "everything I can have".
func ParseModules(raw string, c ProjectConfig) ([]Module, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if raw == "all" {
		var modules []Module
		for _, spec := range AllModules {
			if spec.requires == nil || spec.requires(c) == nil {
				modules = append(modules, spec.name)
			}
		}
		return modules, nil
	}

	var modules []Module
	for _, field := range strings.Split(raw, ",") {
		name := strings.TrimSpace(field)
		if name == "" {
			continue
		}
		if !KnownModule(name) {
			return nil, fmt.Errorf("unknown module %q: choose from %s, or all",
				name, strings.Join(ModuleNames(), ", "))
		}
		modules = append(modules, Module(name))
	}
	return modules, nil
}

// normalizeModules sorts and deduplicates, so a configuration assembled by
// hand scaffolds the same project as one that came from the form.
//
// Determinism is the point: resolveOverlays walks this slice, and an overlay
// order that varied between runs would make two identical configurations
// produce different projects when two overlays touch one file.
func normalizeModules(c *ProjectConfig) {
	if len(c.Modules) == 0 {
		c.Modules = nil
		return
	}

	seen := make(map[Module]bool, len(c.Modules))
	unique := make([]Module, 0, len(c.Modules))
	for _, m := range c.Modules {
		if seen[m] || !KnownModule(string(m)) {
			continue
		}
		seen[m] = true
		unique = append(unique, m)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	c.Modules = unique
}

// validateModules rejects a module the rest of the configuration cannot
// support.
func validateModules(c ProjectConfig) error {
	for _, m := range c.Modules {
		spec, ok := moduleSpecs[m]
		if !ok {
			return fmt.Errorf("unknown module %q: choose from %s", m, strings.Join(ModuleNames(), ", "))
		}
		if spec.requires == nil {
			continue
		}
		if err := spec.requires(c); err != nil {
			return err
		}
	}
	return nil
}

// HasModule reports whether a module was selected.
//
// Takes a string rather than a Module so a template can ask
// {{if .HasModule "mail"}} without the type being addressable from there.
func (c ProjectConfig) HasModule(name string) bool {
	for _, m := range c.Modules {
		if string(m) == name {
			return true
		}
	}
	return false
}

// AnyModule reports whether any optional module was selected, for the
// stubs that only need to know whether to write a block at all.
func (c ProjectConfig) AnyModule() bool { return len(c.Modules) > 0 }

// ModuleImportPath is the Go module path for a selected module, used by
// go.mod.tpl so the pin and the import cannot disagree.
func ModuleImportPath(m Module) string { return "github.com/lemmego/" + string(m) }
