// Package notifications holds this application's notifications, one type per
// kind of thing somebody is told about.
//
// A notification names the channels it goes out on and carries one payload
// per channel. The payload for a channel lives beside that channel's own
// optional interface — ToDatabase for the in-app row, ToMail for the email —
// so this package never learns what an email address is, and a project that
// only wants in-app notifications compiles no SMTP client.
package notifications

import (
	"context"

	"github.com/lemmego/notify"
	dbchan "github.com/lemmego/notify/channel/database"
)

// Welcome is the worked example.
type Welcome struct {
	Name string
}

// Via names the channels, in the order they are delivered.
//
// Returning nil is a legitimate answer rather than an error — somebody who
// has muted this kind gets no channels — so a preference check belongs here.
//
// The order matters when more than one channel is named: database before
// broadcast means the row is committed before anything tells a browser to go
// and look for it. A browser told first refetches and finds nothing.
//
// The names are constants from the channel packages, never string literals,
// so a typo is a compile error rather than a notification delivered nowhere.
func (w Welcome) Via(to notify.Notifiable) []string {
	return []string{dbchan.Name}
}

// NotificationType is the discriminator written to the row.
//
// Spelled out rather than derived from the Go type name, for two reasons:
// renaming this type must not orphan every row already written, and this
// string is an API the frontend switches on to choose a renderer — deriving
// it from an implementation detail makes a refactor a breaking change nobody
// sees until the inbox renders blank rows.
func (w Welcome) NotificationType() string { return "welcome" }

// ToDatabase is the payload for the in-app row.
//
// A struct and not a map, because the JSON is a contract with the client and
// a map is how a field gets quietly renamed.
func (w Welcome) ToDatabase(ctx context.Context, to notify.Notifiable) (any, error) {
	return struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}{
		Name: w.Name,
		Body: "Welcome aboard.",
	}, nil
}

// The compile-time guards. One line each, and they catch the failure this
// design cannot check any other way: a notification whose Via names a
// channel it has no payload method for fails when it is sent, to a real
// person.
var (
	_ notify.Notification = Welcome{}
	_ dbchan.Storable     = Welcome{}
)
