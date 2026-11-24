package routes

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/auth"
	"github.com/lemmego/authz"
	"github.com/lemmego/orm"

	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/lemmego/internal/plugins"
)

// RbacRoutes is the worked example: one route that grants a permission and
// one that checks it.
//
// Behind auth.Protected, because a permission is held by somebody and there
// is nobody to ask about on an anonymous request.
func RbacRoutes(a app.App) {
	a.Router().Post("/rbac/demo", auth.Protected, rbacGrant)
	a.Router().Get("/rbac/demo", auth.Protected, rbacCheck)
}

// rbacGrant creates a role, grants it a permission and assigns the signed-in
// user to it.
//
// Every write takes the transaction, and the scope comes off that transaction
// rather than off an argument — there is no spelling of these calls that
// could name a different scope, which is what makes a cross-tenant write
// unexpressible once this project has tenants.
func rbacGrant(c app.Context) error {
	user, ok := auth.UserAs[*models.User](c)
	if !ok {
		return c.SetStatus(403).JSON(app.M{
			"error": "this endpoint needs a user, not a client credential",
		})
	}

	store := plugins.RbacStore()
	db := app.Get[*orm.DB](c.App())

	err := db.Transaction(c.RequestContext(), func(tx *orm.Tx) error {
		role, err := store.RoleNamed(c.RequestContext(), tx, "editor")
		if err != nil {
			role, err = store.CreateRole(c.RequestContext(), tx, "editor", "Can publish notes")
			if err != nil {
				return err
			}
		}
		if err := store.Grant(c.RequestContext(), tx, role.ID, "note.publish"); err != nil {
			return err
		}
		return store.Assign(c.RequestContext(), tx, user.GetID(), role.ID)
	})
	if err != nil {
		return err
	}

	return c.JSON(app.M{"granted": "note.publish", "role": "editor"})
}

// rbacCheck asks through authz, which is the only way an application should
// ask: the permission set was loaded once for this request and the gate reads
// it off the context.
func rbacCheck(c app.Context) error {
	return c.JSON(app.M{
		// Allowed, not Can: a permission code names an ability with no
		// record attached, and Can is generic over the resource so it cannot
		// infer a type from nil.
		"can_publish": authz.Allowed(c, "note.publish"),
		"can_delete":  authz.Allowed(c, "note.delete"),
	})
}
