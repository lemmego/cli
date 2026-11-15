// Package policies holds the authorization rules, registered at boot.
//
// One place rather than scattered through the handlers, for a reason worth
// stating: a rule written at its call site is a rule nobody can review. With
// them here, "who may edit this" is answerable by reading one file.
package policies

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/authz"
	"github.com/lemmego/lemmego/internal/models"
)

// Abilities are named as "<resource>.<verb>", so they sort by resource.
const (
	ProfileUpdate = "profile.update"
)

// Register is handed the registry at boot.
//
// Registering the same ability twice panics, deliberately: it happens at
// boot, where a panic is a clear message rather than an outage, and the
// alternative — silently taking one of the two — is a rule that is enforced
// on some days and not others depending on file order.
func Register(r *authz.Registry) {
	// The types are inferred from the function, so the body is fully typed:
	// no assertions, no `any`, and a policy that is handed the wrong kind of
	// record does not compile.
	//
	// Ownership compares the record against the signed-in user and costs no
	// extra query — which is the point. A policy that loaded a row to decide
	// whether you may see a row you already have is a query per permission
	// check, and permission checks are not rare.
	authz.Register(r, ProfileUpdate,
		func(c app.Context, user *models.User, target *models.User) authz.Decision {
			if user.ID == target.ID {
				return authz.Allow()
			}
			// The reason reaches the reader, so it is written for them
			// rather than for a log. authz ships translations for its own
			// refusals; this one is the application's and is English until
			// it is given a key.
			return authz.Deny("you can only edit your own profile")
		})
}
