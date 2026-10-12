package cli

import (
	"os"
	"path/filepath"
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

// A generated input binds a form as well as a JSON body.
//
// It used to carry httpin's `in:"form=..."` tags and an *httpin.File, which
// worked only because api routed those structs to httpin — a second binder
// that disagreed with the first about array shapes, about whether a
// checkbox's `on` is true, and about whether files bound at all. api v0.2.0
// has one binder and no httpin, so the generator emits what that binder
// reads.
func TestInputTemplateBindsFormsWithTheFrameworksOwnTags(t *testing.T) {
	stub := inputStub

	if strings.Contains(stub, "httpin") {
		t.Error("input.txt still references httpin, which api no longer depends on")
	}
	if !strings.Contains(stub, `form:"{{.Name | toSnake}}"`) {
		t.Error("input.txt does not emit a form: tag, so a plain HTML form binds nothing")
	}
	if !strings.Contains(stub, `json:"{{.Name | toSnake}}"`) {
		t.Error("input.txt does not emit a json: tag, so a JSON body binds nothing")
	}
}

// A file field is the standard library's type, which carries the filename,
// the size and the headers, and opens to the content.
func TestInputTemplateUsesTheStandardFileType(t *testing.T) {
	stub := inputStub

	if !strings.Contains(stub, "*multipart.FileHeader") {
		t.Error("input.txt does not generate *multipart.FileHeader for a file field")
	}
	if !strings.Contains(stub, `"mime/multipart"`) {
		t.Error("input.txt does not import mime/multipart for a file field")
	}
	// A file cannot survive a JSON round trip, and a struct that claims it
	// can is a struct somebody will try it on.
	if !strings.Contains(stub, "`json:\"-\" form:\"{{.Name | toSnake}}\"`") {
		t.Error("input.txt does not mark a file field json:\"-\"")
	}
}

// Every scaffolded auth input binds a form too.
//
// They carried `json:` tags alone, which is invisible on the Inertia presets
// — those post JSON — and leaves the MVC preset's own sign-in form binding
// nothing at all, with validation reporting "required" for every field the
// person just filled in.
func TestScaffoldedAuthInputsBindForms(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("_scaffold", "overlays", "*", "internal", "inputs", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no scaffolded inputs were found to check")
	}

	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(source)
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, `json:"`) {
				continue
			}
			if strings.Contains(line, `json:"-"`) {
				continue
			}
			if !strings.Contains(line, `form:"`) {
				t.Errorf("%s: %s\n\tbinds JSON but not a form", path, strings.TrimSpace(line))
			}
		}
	}
}
