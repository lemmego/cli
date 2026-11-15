package routes

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
	"github.com/lemmego/authz"
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/lemmego/internal/policies"
)

// AuthzRoutes is the worked example: one route that asks a policy.
//
// Behind auth.Protected, because a policy decides what *this* user may do
// and there is nothing to decide without one. That ordering is the usual
// shape: authentication first, then authorization — a 401 and a 403 are
// different answers and a reader should get the right one.
func AuthzRoutes(a app.App) {
	a.Router().Get("/authz/demo", auth.Protected, authzDemo)
}

func authzDemo(c app.Context) error {
	user, ok := auth.UserAs[*models.User](c)
	if !ok {
		return c.SetStatus(403).JSON(app.M{
			"error": "this endpoint needs a user, not a client credential",
		})
	}

	// Two checks against the same ability with different records, so both
	// answers are visible in one response. A real handler loads the record
	// it is about to change and asks once.
	somebodyElse := &models.User{ID: user.ID + 1}

	// Authorize returns an error carrying the policy's reason, which is what
	// a handler wants: returning it gives the reader the sentence the policy
	// wrote, rather than a bare 403 they cannot act on.
	denied := authz.Authorize(c, policies.ProfileUpdate, somebodyElse)

	return c.JSON(app.M{
		// Can is the boolean form, for deciding whether to render a button.
		// Use it in a view; use Authorize in a handler, where the reason
		// matters.
		"can_edit_self":  authz.Can(c, policies.ProfileUpdate, user),
		"can_edit_other": authz.Can(c, policies.ProfileUpdate, somebodyElse),
		"refusal":        errorText(denied),
	})
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
