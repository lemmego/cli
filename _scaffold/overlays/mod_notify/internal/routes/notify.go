package routes

import (
	"fmt"

	"github.com/lemmego/api/app"
	"github.com/lemmego/lemmego/internal/notifications"
	"github.com/lemmego/lemmego/internal/plugins"
	"github.com/lemmego/notify"
	dbchan "github.com/lemmego/notify/channel/database"
)

// NotifyRoutes is the worked example: notify somebody, then read their inbox.
//
// The recipient is taken from the path so the example runs without
// authentication. A real application takes it from the signed-in user, and
// models.User would implement Notifiable itself rather than needing the
// adapter below.
func NotifyRoutes(a app.App) {
	r := a.Router()
	r.Post("/notify/{id}", sendNotification)
	r.Get("/notify/{id}", listNotifications)
}

// recipient adapts an id to notify's one-method contract.
//
// Strings, because auth's own seam is a string and a project may key its
// users on a bigint or a uuid. The pair is (kind, id) so one table can hold
// notifications for users, teams and anything else without a column per
// type.
type recipient struct{ id string }

func (r recipient) NotifiableKey() (kind, id string) { return "user", r.id }

func sendNotification(c app.Context) error {
	to := recipient{id: c.Param("id")}

	// **Send after the transaction commits, never inside it.** Inside, a
	// broadcast fires before the row is visible and a rollback leaves
	// somebody told about a thing that does not exist. No ordering guarantee
	// inside Send can save a caller from that.
	//
	// It runs inline, which is enough: the only slow channel is mail, and
	// that one renders eagerly and enqueues a self-contained message, so the
	// deferral already happens at the right boundary.
	if err := notify.Send(c.RequestContext(), to, notifications.Welcome{Name: "Ada"}); err != nil {
		return c.SetStatus(502).JSON(app.M{"error": err.Error()})
	}

	return c.JSON(app.M{"notified": c.Param("id")})
}

func listNotifications(c app.Context) error {
	inbox, ok := plugins.Inbox(c.App())
	if !ok {
		return fmt.Errorf("no inbox: add &plugins.NotificationsProvider{} to LoadProviders")
	}

	to := recipient{id: c.Param("id")}

	// Keyset paging, which is the shape an append-only list wants: an offset
	// into a list that is being added to at the top skips and repeats rows.
	// The zero Cursor is the first page.
	rows, next, err := inbox.Page(c.RequestContext(), to, dbchan.Cursor{})
	if err != nil {
		return err
	}

	// A capped count, not a counter column. Unread is per-reader state that
	// changes on read, on mark-all-read and on prune — so a counter column
	// is two sources of truth that drift the first time anybody prunes, and
	// the symptom is a badge reading 3 forever that no log explains.
	unread, err := inbox.Unread(c.RequestContext(), to)
	if err != nil {
		return err
	}

	return c.JSON(app.M{
		"unread":        unread,
		"notifications": rows,
		"next":          next,
	})
}
