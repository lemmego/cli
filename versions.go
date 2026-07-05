package cli

// DependencyVersions pins exact versions for all template dependencies.
// Updated per CLI release to ensure generated projects are reproducible.
var DependencyVersions = map[string]string{
	// Go dependencies
	"github.com/lemmego/api":           "v0.1.24",
	"github.com/lemmego/auth":          "v0.1.4",
	"github.com/lemmego/gormconnector": "v0.1.2",
	"github.com/lemmego/bunconnector":  "v0.1.1",
	"github.com/lemmego/gpa":           "v0.1.1",
	"github.com/lemmego/gpagorm":       "v0.1.5",
	"github.com/lemmego/gpabun":        "v0.1.4",
	"github.com/lemmego/inertia":       "v0.1.2",
	"github.com/lemmego/migration":     "v0.1.14",
	"github.com/lemmego/fsys":          "v0.1.0",
	"github.com/lemmego/cli":           "v0.1.31",
	"github.com/a-h/templ":             "v0.3.943",
	"github.com/spf13/cobra":           "v1.8.1",

	// Node dependencies
	"react":                "^19.0.0",
	"react-dom":            "^19.0.0",
	"@inertiajs/react":     "^3.0.0",
	"vue":                  "^3.5.21",
	"@inertiajs/vue3":      "^3.0.0",
	"@vitejs/plugin-react": "^5.0.2",
	"@vitejs/plugin-vue":   "^6.0.1",
	"laravel-vite-plugin":  "^2.0.1",
	"@tailwindcss/cli":     "^4.3.2",
	"@tailwindcss/vite":    "^4.1.12",
	"tailwindcss":          "^4.1.12",
	"vite":                 "7.1.4",
	"typescript":           "^5.4.5",
	"@types/react":         "^19.0.0",
	"@types/react-dom":     "^19.0.0",
	"@vue/server-renderer": "^3.5.21",
	"axios":                "^1.7.7",
	"@tailwindcss/forms":   "^0.5.7",
}

// ScaffoldVersion is the embedded _scaffold/ directory version.
// The CLI fetches a newer version from GitHub main branch if available.
// Bump this when the scaffold content changes significantly.
const ScaffoldVersion = "1.0.1"
