package plugins

import (
	"github.com/lemmego/api/app"
	"github.com/lemmego/authz"
	"github.com/lemmego/lemmego/internal/policies"
)

// Authz registers this application's policies.
//
// After the auth provider, because a policy's first question is who is
// asking and the answer is resolved by that one.
func Authz() app.Provider {
	return &authz.Provider{Policies: policies.Register}
}
