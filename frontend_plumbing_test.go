package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These files explain the mistakes they avoid, which means they quote the
// wrong idiom in order to warn about it — so a test that grepped the whole
// file would match its own explanation. Comments come out first, and the
// assertion is made against the code.
var (
	lineComment   = regexp.MustCompile(`(?m)//.*$`)
	blockComment  = regexp.MustCompile(`(?s)/\*.*?\*/`)
	goTmplComment = regexp.MustCompile(`(?s)\{\{-?\s*/\*.*?\*/\s*-?\}\}`)
	htmlComment   = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// code strips comments from a TypeScript or TSX source.
func code(source string) string {
	source = blockComment.ReplaceAllString(source, "")
	return lineComment.ReplaceAllString(source, "")
}

// markup strips comments from an HTML file that is also a Go template.
func markup(source string) string {
	source = goTmplComment.ReplaceAllString(source, "")
	return htmlComment.ReplaceAllString(source, "")
}

// reactOverlays are the two presets that ship the React plumbing.
var reactOverlays = []string{
	"_scaffold/overlays/frontend_inertia_react",
	"_scaffold/overlays/frontend_templ_inertia_react",
}

// TestTheLayoutResolverTestsForPresenceNotTruthiness.
//
// This is the one line in the frontend plumbing that is easy to "simplify"
// into a bug. Inertia's convention distinguishes three states on a page
// component:
//
//	nothing said      -> inherit the shell
//	layout = null     -> deliberately none
//	layout = fn       -> its own
//
// `!("layout" in page)` tells the first from the second. `!page.layout` does
// not: it is true for both, so every page that asked for no shell gets one
// anyway — which means a sign-in form rendered inside the signed-in chrome,
// offering a visitor a navigation rail they cannot use.
//
// It compiles, typechecks and looks right, so only this test stands between
// the two.
func TestTheLayoutResolverTestsForPresenceNotTruthiness(t *testing.T) {
	for _, overlay := range reactOverlays {
		path := filepath.Join(overlay, "resources/js/Layouts/resolve.tsx")
		source := code(readFileForTest(t, path))

		if !strings.Contains(source, `"layout" in page`) {
			t.Errorf("%s does not test for the layout property's presence.\n"+
				"A page that exports `layout = null` is asking for no shell, and "+
				"a truthiness check cannot tell that from a page that said "+
				"nothing.", path)
		}
		if strings.Contains(source, "!page.layout") {
			t.Errorf("%s tests page.layout for truthiness, which puts the shell "+
				"back around every page that opted out", path)
		}
	}
}

// TestBothInertiaEntriesApplyTheShell.
//
// The shell has to be applied on both sides. Applying it in app.tsx alone
// means the server renders the page bare and the browser renders it wrapped,
// which is a hydration mismatch — and the symptom is the page visibly
// changing shape just after it loads, which reads as slowness rather than as
// a bug.
func TestBothInertiaEntriesApplyTheShell(t *testing.T) {
	for _, overlay := range reactOverlays {
		for _, entry := range []string{"resources/js/app.tsx", "resources/js/ssr.tsx"} {
			path := filepath.Join(overlay, entry)
			source := readFileForTest(t, path)

			if !strings.Contains(source, "withShell(") {
				t.Errorf("%s does not apply the layout. Both entries have to, or "+
					"the server and the browser render different trees.", path)
			}
		}
	}
}

// TestTheRootTemplateHasExactlyOneTitleSource.
//
// The real <title> is rendered in Go and published by seo.Share, so that it
// survives a server-rendering failure — the kind that answers 200 and looks
// perfect to a human while a crawler receives an empty container.
//
// A static <title> beside it would be a second one on every successful
// render: a browser uses the first and a crawler reports a duplicate. That is
// the bug Components/useDocumentTitle.ts exists to avoid, and putting one back
// in the root template would reintroduce it from the other end.
func TestTheRootTemplateHasExactlyOneTitleSource(t *testing.T) {
	overlays := []string{
		"_scaffold/overlays/frontend_inertia_react",
		"_scaffold/overlays/frontend_templ_inertia_react",
		"_scaffold/overlays/frontend_inertia_vue",
		"_scaffold/overlays/frontend_templ_inertia_vue",
	}

	for _, overlay := range overlays {
		path := filepath.Join(overlay, "resources/views/root.html")
		source := markup(readFileForTest(t, path))

		if !strings.Contains(source, "{{ .seoHead }}") {
			t.Errorf("%s does not render the head published by seo.Share, so a "+
				"project with lemmego/seo emits no title, no canonical URL and no "+
				"structured data", path)
		}
		if strings.Contains(source, "<title>") {
			t.Errorf("%s carries a static <title> as well as the head seo "+
				"renders, which is two <title> tags in one document", path)
		}
	}
}

// TestTheBroadcastClientInTheOverlayImportsNoFramework.
//
// mod_broadcast applies to every project that selects the module, including a
// Vue one and one with no frontend at all — and tsconfig.json typechecks
// everything under resources/js. A React import in that overlay fails to
// resolve in a Vue project, which is a typecheck error in a file nobody
// touched.
//
// The React hook is therefore a stub, written only when the frontend is
// React. See generateDynamicFiles.
func TestTheBroadcastClientInTheOverlayImportsNoFramework(t *testing.T) {
	dir := "_scaffold/overlays/mod_broadcast/resources/js/lib"

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}

	var sawClient bool
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".ts") {
			continue
		}
		sawClient = true
		source := code(readFileForTest(t, filepath.Join(dir, entry.Name())))

		for _, framework := range []string{`from "react"`, `from "vue"`} {
			if strings.Contains(source, framework) {
				t.Errorf("%s/%s imports %s, and this overlay also applies to "+
					"projects built with the other one — and to projects with no "+
					"frontend at all", dir, entry.Name(), framework)
			}
		}
	}
	if !sawClient {
		t.Errorf("%s ships no TypeScript client, so a project selecting "+
			"broadcast has nothing to subscribe with", dir)
	}
}

// TestTheReactHookIsWrittenOnlyForAReactFrontend is the other half: the stub
// exists, and it is gated.
func TestTheReactHookIsWrittenOnlyForAReactFrontend(t *testing.T) {
	stub := "_scaffold/stubs/use_broadcast.ts.tpl"
	if _, err := os.Stat(stub); err != nil {
		t.Fatalf("%s is missing, so a React project selecting broadcast gets no hook: %v", stub, err)
	}

	engine := readFileForTest(t, "scaffold_engine.go")
	if !strings.Contains(engine, `td.HasModule("broadcast") && td.FrontendHasReact`) {
		t.Error("the broadcast hook is not gated on a React frontend, so a Vue " +
			"project would be given a file importing React")
	}
}

// TestTheTwoReactOverlaysShipTheSameFrontend.
//
// inertia_react and templ_inertia_react differ in their Go templates and in
// nothing else: the JavaScript is the same application. They were identical
// before this plumbing existed, and a fix applied to one and not the other is
// the kind of drift nobody notices until somebody generates the less-used
// preset.
func TestTheTwoReactOverlaysShipTheSameFrontend(t *testing.T) {
	left := filepath.Join(reactOverlays[0], "resources/js")
	right := filepath.Join(reactOverlays[1], "resources/js")

	files := map[string]bool{}
	for _, root := range []string{left, right} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil {
				files[rel] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}

	for name := range files {
		a, errA := os.ReadFile(filepath.Join(left, name))
		b, errB := os.ReadFile(filepath.Join(right, name))

		switch {
		case errA != nil:
			t.Errorf("%s is in %s and not in %s", name, right, left)
		case errB != nil:
			t.Errorf("%s is in %s and not in %s", name, left, right)
		case string(a) != string(b):
			t.Errorf("resources/js/%s differs between the two React overlays.\n"+
				"They ship the same application; a change belongs in both.", name)
		}
	}
}

// TestNoScaffoldedReactFileImportsReactForJsx.
//
// react-jsx needs no import, and under this tsconfig an unused one is a
// typecheck error rather than a harmless line. The existing contract test
// covers the two entry points; this covers everything the plumbing added,
// where the same mistake is easier to make because these files are ordinary
// components.
func TestNoScaffoldedReactFileImportsReactForJsx(t *testing.T) {
	for _, overlay := range reactOverlays {
		root := filepath.Join(overlay, "resources/js")

		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if !strings.HasSuffix(path, ".tsx") && !strings.HasSuffix(path, ".ts") {
				return nil
			}

			source := code(readFileForTest(t, path))
			if strings.Contains(source, `import React from "react"`) {
				t.Errorf("%s imports React for JSX, which react-jsx does not need "+
					"and noUnusedLocals rejects", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}

// TestTheShellRaisesFlashMessages.
//
// The bridge is what makes a redirect-with-a-flash visible without every
// handler remembering to render one. It is called in the shell rather than in
// each page for the same reason the shell is applied in resolve.tsx: a page
// that forgets is a message nobody sees, and nothing fails.
func TestTheShellRaisesFlashMessages(t *testing.T) {
	for _, overlay := range reactOverlays {
		shell := readFileForTest(t, filepath.Join(overlay, "resources/js/Layouts/AppShell.tsx"))

		if !strings.Contains(shell, "useFlashToasts()") {
			t.Errorf("%s/resources/js/Layouts/AppShell.tsx does not raise flash "+
				"messages, so a handler that redirects with one shows nothing", overlay)
		}
		if !strings.Contains(shell, "<Toasts />") {
			t.Errorf("%s/resources/js/Layouts/AppShell.tsx does not render the "+
				"toast stack, so every toast raised anywhere is invisible", overlay)
		}
	}
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(raw)
}
