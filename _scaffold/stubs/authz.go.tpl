package plugins

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/authz"
{{- if .HasModule "authz"}}
	"github.com/lemmego/lemmego/internal/policies"
{{- end}}
{{- if .HasModule "rbac"}}
	"github.com/lemmego/lemmego/internal/models"
	"github.com/lemmego/rbac/rbacauthz"
{{- end}}
)

// Authz registers this application's authorization rules.
//
// After the auth provider, because a policy's first question is who is asking
// and the answer is resolved by that one.
//
// There is one authz provider however many sources of rules there are: typed
// policies and database-backed permissions both register into the same
// registry, and a second provider would overwrite the first's registrations
// rather than adding to them.
func Authz() app.Provider {
	return &authz.Provider{Policies: func(r *authz.Registry) {
{{- if .HasModule "authz"}}
		// Typed policies: Go code, reviewable in one place, changed by
		// editing this project.
		policies.Register(r)
{{- end}}
{{- if .HasModule "rbac"}}
		// Database-backed roles and permissions. Install registers one gate
		// per declared permission code, plus one Before hook for the system
		// role — and both are needed. With the gates alone a member of the
		// system role passes every permission check and is still refused by
		// an ownership policy on somebody else's row.
		rbacauthz.Install[*models.User](r, RbacCatalog(), rbacauthz.Options{})
{{- end}}
	}}
}
