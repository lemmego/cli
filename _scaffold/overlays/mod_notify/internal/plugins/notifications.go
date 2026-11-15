package plugins

import (
	"fmt"

	"github.com/lemmego/api/app"
	"github.com/lemmego/api/db"
	"github.com/lemmego/notify"
	dbchan "github.com/lemmego/notify/channel/database"
	"github.com/lemmego/notify/channel/database/sqlstore"
	nprovider "github.com/lemmego/notify/provider"
)

// NotificationsProvider wires notify's channels to this application's
// services.
//
// A provider of this application's rather than notify's own, and the reason
// is which way the dependencies point: notify declares the channel contract
// and knows nothing about this project's database, while this file knows
// both. The store needs the connection, which is why this cannot be a
// literal in LoadProviders — that list is written before the container
// exists.
type NotificationsProvider struct{}

func (p *NotificationsProvider) Provide(a app.App) error {
	conn, ok := db.Resolve(a)
	if !ok {
		return fmt.Errorf("notify: no database connection, so the in-app " +
			"channel has nowhere to write. Register the database connector " +
			"before this provider.")
	}

	// Takes the application's own pool — one pool, one set of limits —
	// rather than opening a second one whose size nobody configured.
	store, err := sqlstore.New(conn, sqlstore.WithTable(sqlstore.DefaultTable))
	if err != nil {
		return fmt.Errorf("notify: building the store: %w", err)
	}

	channels := []notify.Channel{dbchan.New(store)}

	// The mail and broadcast channels are one line each, and both are
	// deliberately absent here rather than wired and unused:
	//
	//	channels = append(channels, mailchan.New(mailer, mailchan.WithQueue(true)))
	//	channels = append(channels, bcast.New(publisher))
	//
	// Email for every notification is spam, and the preference system that
	// would make it defensible is a feature of its own — so the honest
	// default is in-app only, and mail for the one or two kinds somebody has
	// to act on.

	if err := (&nprovider.Provider{Channels: channels}).Provide(a); err != nil {
		return err
	}

	// The inbox reader, for the routes that list and mark notifications.
	// Registered here because it shares the store the channel writes
	// through, and two stores over one table would be two caches of one
	// truth.
	a.AddService(dbchan.NewInbox(store))
	return nil
}

// Inbox resolves the reader.
//
// Lookup rather than Get, because Get panics when a service is absent and
// this is read while rendering — so a project that has not wired the
// provider would lose every page rather than one feature.
func Inbox(a app.App) (*dbchan.Inbox, bool) {
	return app.Lookup[*dbchan.Inbox](a)
}
