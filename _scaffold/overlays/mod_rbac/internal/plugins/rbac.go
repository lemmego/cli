package plugins

import (
	"github.com/lemmego/rbac"
	"github.com/lemmego/rbac/rbacstore"
)

// RbacCatalog is every permission this build declares.
//
// A value this application owns rather than a registry filled by init
// functions: a global one works until a test wants two, and then one test's
// declarations satisfy another's assertions.
//
// Add a permission here and it becomes checkable as
// authz.Authorize(c, "post.publish", nil). A code nobody declares is denied —
// authz refuses an unregistered ability before any hook runs — so a typo fails
// closed rather than passing for whoever happens to be an administrator.
func RbacCatalog() *rbac.Catalog {
	catalog := rbac.NewCatalog()

	// Two to start with, so the shape is visible. A code is "module.action",
	// lower case, and Module must be its first segment; both are checked
	// here rather than at the call site.
	if err := catalog.Add(
		rbac.Permission{
			Code:        "note.publish",
			Module:      "note",
			Description: "Make a note visible to everybody.",
		},
		rbac.Permission{
			Code:        "note.delete",
			Module:      "note",
			Description: "Remove a note somebody else wrote.",
		},
	); err != nil {
		// A malformed code is a programming error in this file, fixed by
		// editing it, so failing loudly at boot is the right answer.
		panic("rbac catalogue: " + err.Error())
	}

	return catalog
}

// RbacStore reads and writes the roles, grants and assignments.
//
// Unscoped(""), because this project is not multi-tenant. The default instead
// reads the tenant the ORM transaction is bound to, and leaving it as the
// default here would answer ErrNoScope on the first call.
//
// If you add lemmego/tenancy later, delete the ScopeOf line: the default is
// then the one you want, and every role becomes per-tenant without changing
// any call site.
func RbacStore() *rbacstore.Store {
	return rbacstore.New(rbacstore.Config{
		ScopeOf: rbacstore.Unscoped(""),
	})
}
