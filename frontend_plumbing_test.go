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

// code strips comments from a TypeScript, TSX, JavaScript or Vue source. A
// single-file component is both at once, which is why the HTML form is
// stripped here too.
//
// It is a regexp and not a parser, so it also eats the tail of any line
// containing "//" — a URL in a template attribute, for one. That is fine for
// every assertion below and would not be fine for an assertion about a URL,
// which would pass having matched nothing.
func code(source string) string {
	source = blockComment.ReplaceAllString(source, "")
	source = htmlComment.ReplaceAllString(source, "")
	return lineComment.ReplaceAllString(source, "")
}

// markup strips comments from an HTML file that is also a Go template.
func markup(source string) string {
	source = goTmplComment.ReplaceAllString(source, "")
	return htmlComment.ReplaceAllString(source, "")
}

// The four Inertia overlays, in the two pairs that ship the same frontend.
//
// Each pair differs in its Go templates and in nothing else: the JavaScript is
// the same application, which is what TestEachOverlayPairShipsTheSameFrontend
// is for.
var (
	reactOverlays = []string{
		"_scaffold/overlays/frontend_inertia_react",
		"_scaffold/overlays/frontend_templ_inertia_react",
	}
	vueOverlays = []string{
		"_scaffold/overlays/frontend_inertia_vue",
		"_scaffold/overlays/frontend_templ_inertia_vue",
	}
	inertiaOverlays = append(append([]string{}, reactOverlays...), vueOverlays...)
)

// plumbing names, per framework, the files whose paths differ between the two.
type plumbing struct {
	overlays    []string
	resolver    string // the layout resolver
	shell       string // the application shell
	clientEntry string
	ssrEntry    string
}

var frameworks = map[string]plumbing{
	"react": {
		overlays:    reactOverlays,
		resolver:    "resources/js/Layouts/resolve.tsx",
		shell:       "resources/js/Layouts/AppShell.tsx",
		clientEntry: "resources/js/app.tsx",
		ssrEntry:    "resources/js/ssr.tsx",
	},
	"vue": {
		overlays:    vueOverlays,
		resolver:    "resources/js/Layouts/resolve.ts",
		shell:       "resources/js/Layouts/AppShell.vue",
		clientEntry: "resources/js/app.js",
		ssrEntry:    "resources/js/ssr.js",
	},
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
// the two. It holds for Vue as well, where the adapter's own check on
// `component.layout` is a truthiness test — which is exactly why ours has to
// be a presence test, or `null` never reaches it.
func TestTheLayoutResolverTestsForPresenceNotTruthiness(t *testing.T) {
	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			path := filepath.Join(overlay, fw.resolver)
			source := code(readFileForTest(t, path))

			if !strings.Contains(source, `"layout" in page`) {
				t.Errorf("%s (%s) does not test for the layout property's presence.\n"+
					"A page that exports `layout = null` is asking for no shell, and "+
					"a truthiness check cannot tell that from a page that said "+
					"nothing.", path, name)
			}
			if strings.Contains(source, "!page.layout") {
				t.Errorf("%s (%s) tests page.layout for truthiness, which puts the "+
					"shell back around every page that opted out", path, name)
			}
		}
	}
}

// TestTheVueOptOutIsOneVueTscAccepts.
//
// The React overlay's `export const layout = null` is Inertia's own
// convention and typechecks there, because nothing types a React page's
// exports. In Vue it does not: @inertiajs/vue3 augments Vue's component
// options with a `layout` whose type does not admit `null`, so a page written
// that way fails `vue-tsc` — and a documented opt-out that does not typecheck
// is worse than none, because it is found by the first person who uses it.
//
// `noShell` is `undefined` under a name, which the type admits and which the
// presence check still reads as "deliberately none".
func TestTheVueOptOutIsOneVueTscAccepts(t *testing.T) {
	for _, overlay := range vueOverlays {
		path := filepath.Join(overlay, frameworks["vue"].resolver)
		source := readFileForTest(t, path)

		if !strings.Contains(code(source), "export const noShell = undefined;") {
			t.Errorf("%s exports no opt-out constant, so a Vue page has no way to "+
				"decline the shell that typechecks", path)
		}
		if strings.Contains(code(source), "layout: null") {
			t.Errorf("%s offers `layout: null` as the opt-out, which is a vue-tsc "+
				"error: @inertiajs/vue3 types this option without null", path)
		}
	}
}

// TestBothInertiaEntriesApplyTheShell.
//
// The shell has to be applied on both sides. Applying it in the client entry
// alone means the server renders the page bare and the browser renders it
// wrapped, which is a hydration mismatch — and the symptom is the page visibly
// changing shape just after it loads, which reads as slowness rather than as
// a bug.
func TestBothInertiaEntriesApplyTheShell(t *testing.T) {
	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			for _, entry := range []string{fw.clientEntry, fw.ssrEntry} {
				path := filepath.Join(overlay, entry)
				source := readFileForTest(t, path)

				if !strings.Contains(source, "withShell(") {
					t.Errorf("%s (%s) does not apply the layout. Both entries have "+
						"to, or the server and the browser render different trees.",
						path, name)
				}
			}
		}
	}
}

// TestNothingRaisesTheFlashTwice.
//
// The flash that arrives with the document is announced by Inertia's event
// like any other, which is not the obvious answer: the page is written into
// history before being announced and history carries no flash, deliberately,
// so that going back does not repeat a message. The announcement reads the
// page it was handed rather than the one that came back out, so the message
// survives — measured over eight document loads, each raising exactly one
// toast.
//
// So raising `props.initialPage.flash` in the entry's setup, which is the
// obvious belt and braces and which this plumbing briefly did, shows every
// such message twice. In a server renderer it is worse than that: setup runs
// there too, and the queue is module-level.
func TestNothingRaisesTheFlashTwice(t *testing.T) {
	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			for _, entry := range []string{fw.clientEntry, fw.ssrEntry} {
				path := filepath.Join(overlay, entry)
				source := code(readFileForTest(t, path))

				if strings.Contains(source, "initialPage.flash") {
					t.Errorf("%s (%s) raises the document's flash as well as "+
						"listening for it, which shows it twice", path, name)
				}
			}
		}
	}
}

// TestTheClientEntryWiresTheFlashBridge.
//
// In the entry rather than in a component, and in the client entry alone. The
// listener is registered for the life of the tab; a component that registered
// one would do it per mount, and a server renderer would do it per render.
func TestTheClientEntryWiresTheFlashBridge(t *testing.T) {
	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			path := filepath.Join(overlay, fw.clientEntry)
			source := code(readFileForTest(t, path))

			if !strings.Contains(source, "listenForFlash()") {
				t.Errorf("%s (%s) does not call listenForFlash(), so a handler that "+
					"redirects with a flash message shows nothing", path, name)
			}
			if !strings.Contains(source, "listenForFlash();\n\ncreateInertiaApp") {
				t.Errorf("%s (%s) does not register the flash listener before "+
					"createInertiaApp, so the message that arrived with this "+
					"document — announced during initialisation — reaches nobody",
					path, name)
			}
		}
	}
}

// TestNothingReadsTheFlashFromThePage.
//
// The bug this plumbing shipped with, pinned.
//
// The flash is a real field of the page object and it does arrive in the
// document — so `usePage().flash` looks right, typechecks, and works in a
// browser if you happen to read it early enough. Inertia's core blanks it on
// the page it hands out, deliberately, so that going back in history does not
// re-announce something that already happened. That made the React version
// read an empty object and the Vue version read the real one, from identical
// code. `page.props.flash` is worse again: props is an open index signature,
// so it typechecks and is undefined forever.
func TestNothingReadsTheFlashFromThePage(t *testing.T) {
	for _, overlay := range inertiaOverlays {
		root := filepath.Join(overlay, "resources/js")

		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			source := code(readFileForTest(t, path))
			for _, wrong := range []string{"page.flash", "props.flash", "page).flash"} {
				if strings.Contains(source, wrong) {
					t.Errorf("%s reads %s. The flash reaches the browser through "+
						"Inertia's flash event, not through the page — see "+
						"resources/js/lib/flash.ts.", path, wrong)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
}

// TestTheShellRendersTheToastStack.
//
// Rendered in the shell rather than in each page for the same reason the shell
// is applied in the resolver: a page that forgets is a message nobody sees,
// and nothing fails.
func TestTheShellRendersTheToastStack(t *testing.T) {
	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			path := filepath.Join(overlay, fw.shell)
			shell := code(readFileForTest(t, path))

			if !strings.Contains(shell, "<Toasts />") {
				t.Errorf("%s (%s) does not render the toast stack, so every toast "+
					"raised anywhere is invisible", path, name)
			}
		}
	}
}

// TestTheFrameworkFreeFilesAreOneFileEverywhere.
//
// lib/toast.ts and lib/flash.ts import no framework, which is the reason they
// are the same bytes in all four overlays. Keeping them identical is what
// makes "fix it in one place" true rather than aspirational — and a divergence
// here is the shape of bug that gets found in the preset nobody generates.
func TestTheFrameworkFreeFilesAreOneFileEverywhere(t *testing.T) {
	for _, name := range []string{"resources/js/lib/toast.ts", "resources/js/lib/flash.ts"} {
		want := readFileForTest(t, filepath.Join(inertiaOverlays[0], name))
		for _, overlay := range inertiaOverlays[1:] {
			got := readFileForTest(t, filepath.Join(overlay, name))
			if got != want {
				t.Errorf("%s differs between %s and %s. It imports no framework, so "+
					"there is no reason for two versions of it.",
					name, inertiaOverlays[0], overlay)
			}
		}
		source := code(want)
		for _, framework := range []string{`from "react"`, `from "vue"`, `@inertiajs/react`, `@inertiajs/vue3`} {
			if strings.Contains(source, framework) {
				t.Errorf("%s imports %s, so it cannot be the same file in every "+
					"overlay", name, framework)
			}
		}
	}
}

// TestNeitherFrameworksPlumbingImportsTheOther.
//
// These two trees are written by copying from each other, which is how a React
// import ends up in a Vue overlay — where it does not resolve, and the
// typecheck fails in a file nobody touched.
func TestNeitherFrameworksPlumbingImportsTheOther(t *testing.T) {
	forbidden := map[string][]string{
		"react": {`from "vue"`, `@inertiajs/vue3`},
		"vue":   {`from "react"`, `@inertiajs/react`, `react-dom`},
	}

	for name, fw := range frameworks {
		for _, overlay := range fw.overlays {
			root := filepath.Join(overlay, "resources/js")

			err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				source := code(readFileForTest(t, path))
				for _, bad := range forbidden[name] {
					if strings.Contains(source, bad) {
						t.Errorf("%s is in the %s overlay and imports %s", path, name, bad)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walking %s: %v", root, err)
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
// the bug Components/useDocumentTitle exists to avoid, and putting one back
// in the root template would reintroduce it from the other end.
func TestTheRootTemplateHasExactlyOneTitleSource(t *testing.T) {
	for _, overlay := range inertiaOverlays {
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
// mod_broadcast applies to every project that selects the module, including
// one with no frontend at all — and tsconfig.json typechecks everything under
// resources/js. A React import in that overlay fails to resolve in a Vue
// project, which is a typecheck error in a file nobody touched.
//
// The binding is therefore a stub, written according to the frontend the
// project chose. See generateDynamicFiles.
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

// TestTheBroadcastBindingIsGatedOnTheFrontend is the other half: a stub per
// framework, each gated, both written to the same path so an application
// imports ./useBroadcast either way.
func TestTheBroadcastBindingIsGatedOnTheFrontend(t *testing.T) {
	for _, stub := range []string{
		"_scaffold/stubs/use_broadcast.ts.tpl",
		"_scaffold/stubs/use_broadcast_vue.ts.tpl",
	} {
		if _, err := os.Stat(stub); err != nil {
			t.Fatalf("%s is missing, so a project selecting broadcast gets no "+
				"binding for its frontend: %v", stub, err)
		}
	}

	engine := readFileForTest(t, "scaffold_engine.go")
	for _, gate := range []string{
		`td.HasModule("broadcast") && td.FrontendHasReact`,
		`td.HasModule("broadcast") && td.FrontendHasVue`,
	} {
		if !strings.Contains(engine, gate) {
			t.Errorf("the broadcast binding is not gated on %q, so a project "+
				"would be given a file importing the framework it does not have", gate)
		}
	}
}

// TestEachOverlayPairShipsTheSameFrontend.
//
// inertia_<fw> and templ_inertia_<fw> differ in their Go templates and in
// nothing else: the JavaScript is the same application. They were identical
// before this plumbing existed, and a fix applied to one and not the other is
// the kind of drift nobody notices until somebody generates the less-used
// preset.
func TestEachOverlayPairShipsTheSameFrontend(t *testing.T) {
	for name, fw := range frameworks {
		left := filepath.Join(fw.overlays[0], "resources/js")
		right := filepath.Join(fw.overlays[1], "resources/js")

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

		for file := range files {
			a, errA := os.ReadFile(filepath.Join(left, file))
			b, errB := os.ReadFile(filepath.Join(right, file))

			switch {
			case errA != nil:
				t.Errorf("%s is in %s and not in %s", file, right, left)
			case errB != nil:
				t.Errorf("%s is in %s and not in %s", file, left, right)
			case string(a) != string(b):
				t.Errorf("resources/js/%s differs between the two %s overlays.\n"+
					"They ship the same application; a change belongs in both.", file, name)
			}
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

// TestTheVuePresetTypechecksWithVueTsc.
//
// A Vue project's typecheck is `vue-tsc`, and that is not a preference. Plain
// `tsc` has no idea what a .vue file is: every import of one is reported as a
// missing module and nothing inside a <script setup> block is read at all.
// Vite never typechecks either — esbuild strips the types — so without the
// right tool pinned and named, a type error in a component builds and ships,
// which is how a scaffolded project failed its own typecheck on the first
// file for two releases.
func TestTheVuePresetTypechecksWithVueTsc(t *testing.T) {
	manifest := readFileForTest(t, "_scaffold/stubs/package.json.tpl")

	if !strings.Contains(manifest, `"typecheck": "vue-tsc --noEmit"`) {
		t.Error("package.json.tpl ships no vue-tsc typecheck script for a Vue project")
	}
	if !strings.Contains(manifest, `"vue-tsc": "{{.Version "vue-tsc"}}"`) {
		t.Error("vue-tsc is not a dependency of a generated Vue project, so the " +
			"typecheck script cannot run")
	}
	if _, ok := scaffoldPins(t)["vue-tsc"]; !ok {
		t.Error("vue-tsc has no pin in versions.json, so Version answers " +
			"\"latest\" and the manifest asks for a version npm cannot resolve")
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
