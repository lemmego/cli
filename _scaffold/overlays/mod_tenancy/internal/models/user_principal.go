package models

// PrincipalKind and PrincipalKey make *User a tenancy.Principal, so the
// tenancy provider can ask whether the signed-in person belongs to the tenant
// a request names.
//
// In its own file rather than user.go, so regenerating the model does not
// lose it.
//
// Kind exists because membership is a different query for each population: a
// bare key would be ambiguous, and user 42 and sysadmin 42 are different
// people. Return "sysadmin" from a separate type if you add platform staff
// who can reach every tenant; do not overload this one.
func (u *User) PrincipalKind() string { return "user" }

// PrincipalKey identifies the user within that population. GetID rather than
// the raw column, so the key is a string whatever the model's key type is.
func (u *User) PrincipalKey() string { return u.GetID() }
