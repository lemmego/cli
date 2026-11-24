package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"testing"
)

// TestEveryDatabaseWrapsItsOwnDriverForObserve guards the one intersection the
// compile gate's matrix does not reach.
//
// observe_driver.go is a stub rather than an overlay file because the driver it
// wraps is the one the project chose, and the template is a chain of else-if
// branches with no default. A database with no branch generates
// `observe.Wrap()` — no import, no argument — which is a type error, so the
// project does not build at all. libSQL had no branch.
//
// The gate missed it because its axes never cross: the configs that select
// modules use postgres and sqlite, and the configs that use libSQL select no
// modules, so no combination ever emitted this file for libSQL.
//
// Parsed rather than grepped, because the broken output also contains the text
// "observe.Wrap(". What matters is that the call has an argument and that the
// package it names is imported, and only the syntax tree answers that.
func TestEveryDatabaseWrapsItsOwnDriverForObserve(t *testing.T) {
	for _, driver := range []SQLDriver{SQLSQLite, SQLMySQL, SQLPostgres, SQLLibSQL} {
		t.Run(string(driver), func(t *testing.T) {
			dir := scaffoldForContract(t, ProjectConfig{
				ModuleName: "example.com/observe/" + string(driver),
				Preset:     PresetMVC,
				SQLDriver:  driver,
				Modules:    []Module{"observe"},
			})
			src := readGenerated(t, dir, "internal/plugins/observe_driver.go")

			file, err := parser.ParseFile(token.NewFileSet(), "observe_driver.go", src, 0)
			if err != nil {
				t.Fatalf("the generated file does not parse: %v\n%s", err, src)
			}

			// Every package this file can refer to, by the name the code uses:
			// the explicit alias when there is one, the last path element
			// otherwise.
			imported := map[string]bool{}
			for _, spec := range file.Imports {
				name := path.Base(spec.Path.Value[1 : len(spec.Path.Value)-1])
				if spec.Name != nil {
					name = spec.Name.Name
				}
				imported[name] = true
			}

			wraps := 0
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Wrap" {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "observe" {
					return true
				}
				wraps++

				if len(call.Args) != 1 {
					t.Errorf("observe.Wrap takes %d arguments, want 1: %s has no branch "+
						"in observe_driver.go.tpl, so it wraps nothing and the generated "+
						"project does not compile", len(call.Args), driver)
					return true
				}
				pkg := packageOf(call.Args[0])
				if pkg == "" {
					t.Errorf("observe.Wrap is passed %T, want a driver value from a "+
						"driver package", call.Args[0])
					return true
				}
				if !imported[pkg] {
					t.Errorf("observe.Wrap is passed a %s value but %s is not imported, "+
						"so %s got a driver branch without the matching import branch",
						pkg, pkg, driver)
				}
				return true
			})

			if wraps != 1 {
				t.Fatalf("found %d calls to observe.Wrap, want exactly 1:\n%s", wraps, src)
			}
		})
	}
}

// packageOf names the package qualifying a driver value, for both &pkg.T{} and
// pkg.T{}. It returns "" for anything else.
func packageOf(arg ast.Expr) string {
	if unary, ok := arg.(*ast.UnaryExpr); ok {
		arg = unary.X
	}
	lit, ok := arg.(*ast.CompositeLit)
	if !ok {
		return ""
	}
	sel, ok := lit.Type.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return pkg.Name
}
