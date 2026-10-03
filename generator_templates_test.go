package cli

import (
	"strings"
	"testing"
)

// Both of these generators have been emitting code that does not compile.
// A generator that produces a broken file is worse than no generator: the
// error surfaces later, somewhere else, and reads like the developer's
// mistake.
func TestHandlerTemplateUsesTheHandlerSignature(t *testing.T) {
	// app.Context is an interface. *app.Context is a pointer to an interface,
	// which does not satisfy app.Handler and is almost never what anyone
	// means.
	if strings.Contains(handlerStub, "*app.Context") {
		t.Error("handler.txt generates ctx *app.Context, a pointer to an interface, which does not satisfy app.Handler")
	}
	if !strings.Contains(handlerStub, "ctx app.Context") {
		t.Error("handler.txt does not generate the app.Handler signature")
	}
}

func TestInputTemplateUsesARuleThatExists(t *testing.T) {
	stub := inputStub

	// vField has no Unique method and never has. The rule it needs lives in
	// api/db, reached through vField.Check.
	if strings.Contains(stub, ").Unique(") {
		t.Error("input.txt generates .Unique(...), which does not exist on vField")
	}
	if !strings.Contains(stub, "db.Unique(") {
		t.Error("input.txt does not generate the db.Unique rule")
	}
	if !strings.Contains(stub, "Check(") {
		t.Error("input.txt does not use vField.Check, which is how a rule that can fail reports it")
	}
	// A uniqueness lookup that fails is not the same as a value that is
	// taken, and the generated input has to distinguish them or it tells
	// someone their value is in use when the database is unreachable.
	if !strings.Contains(stub, "v.Err()") {
		t.Error("input.txt does not check Validator.Err, so a failed lookup is reported as a taken value")
	}
	// api/db must be imported only when it is used; an unused import does not
	// compile.
	if !strings.Contains(stub, "{{- if .HasUnique}}") {
		t.Error("input.txt imports api/db unconditionally, which does not compile for an input with no unique field")
	}
}

func TestHasUniqueReportsWhetherAnyFieldNeedsTheImport(t *testing.T) {
	if (InputConfig{Fields: []*InputField{{Name: "name"}}}).HasUnique() {
		t.Error("HasUnique reported true with no unique field")
	}
	if !(InputConfig{Fields: []*InputField{{Name: "name"}, {Name: "email", Unique: true}}}).HasUnique() {
		t.Error("HasUnique reported false with a unique field")
	}
	if (InputConfig{}).HasUnique() {
		t.Error("HasUnique reported true for no fields at all")
	}
}
