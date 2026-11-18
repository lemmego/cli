package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

const frontendManifestName = "lemmego.frontend.json"
const frontendGeneratedPath = ".lemmego/frontend.generated.json"

type frontendManifest struct {
	Schema           int               `json:"schema"`
	ID               string            `json:"id"`
	Framework        string            `json:"framework"`
	Pages            map[string]string `json:"pages"`
	Entries          map[string]string `json:"entries,omitempty"`
	Styles           []string          `json:"styles,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
}

type frontendPackage struct {
	ImportPath string `json:"ImportPath"`
	Dir        string `json:"Dir"`
	Standard   bool   `json:"Standard"`
	DepOnly    bool   `json:"DepOnly"`
}

type frontendContribution struct {
	ID               string            `json:"id"`
	Framework        string            `json:"framework"`
	Package          string            `json:"package"`
	Root             string            `json:"root"`
	Pages            map[string]string `json:"pages,omitempty"`
	Entries          map[string]string `json:"entries,omitempty"`
	Styles           []string          `json:"styles,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
}

type frontendBuildManifest struct {
	Schema  int                    `json:"schema"`
	Pages   map[string]string      `json:"pages"`
	Modules []frontendContribution `json:"modules"`
}

var frontendCmd = &cobra.Command{
	Use:   "frontend",
	Short: "Manage package-owned frontend contributions",
}

var frontendSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Discover frontend contributions from compiled Go packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := syncFrontend(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Frontend contributions written to %s.\n", frontendGeneratedPath)
		return nil
	},
}

func init() {
	frontendCmd.AddCommand(frontendSyncCmd)
}

func syncFrontend() error {
	output, err := runCommand(".", "go", "list", "-deps", "-json", "./cmd/app")
	if err != nil {
		return fmt.Errorf("discovering Go packages: %w", err)
	}

	packages, err := decodeFrontendPackages(output)
	if err != nil {
		return fmt.Errorf("reading Go package list: %w", err)
	}
	manifest, err := buildFrontendManifest(packages)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding frontend manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(frontendGeneratedPath), 0o755); err != nil {
		return fmt.Errorf("creating frontend metadata directory: %w", err)
	}
	if existing, err := os.ReadFile(frontendGeneratedPath); err == nil && bytes.Equal(existing, data) {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(frontendGeneratedPath), ".frontend-generated-*")
	if err != nil {
		return fmt.Errorf("staging frontend manifest: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing frontend manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing frontend manifest: %w", err)
	}
	if err := os.Rename(tmpName, frontendGeneratedPath); err != nil {
		return fmt.Errorf("installing frontend manifest: %w", err)
	}
	return nil
}

func syncFrontendIfProject() {
	if !fileExists("go.mod") || !dirExists("cmd/app") || (!hasFrontendManifestFiles() && !hasLemmegoFrontendPackage()) {
		return
	}
	fmt.Println("> Discovering package frontend contributions...")
	if err := syncFrontend(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func hasLemmegoFrontendPackage() bool {
	data, err := os.ReadFile("package.json")
	return err == nil && bytes.Contains(data, []byte(`"@lemmego/vite"`))
}

func hasFrontendManifestFiles() bool {
	found := false
	_ = filepath.WalkDir(".", func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if current != "." && (filepath.Base(current) == ".git" || filepath.Base(current) == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(current) == frontendManifestName {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func decodeFrontendPackages(output []byte) ([]frontendPackage, error) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []frontendPackage
	for {
		var pkg frontendPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func buildFrontendManifest(packages []frontendPackage) (frontendBuildManifest, error) {
	result := frontendBuildManifest{Schema: 1, Pages: map[string]string{}, Modules: []frontendContribution{}}
	seenPackages := map[string]bool{}
	seenIDs := map[string]bool{}
	for _, pkg := range packages {
		if pkg.Standard || pkg.Dir == "" || seenPackages[pkg.ImportPath] {
			continue
		}
		seenPackages[pkg.ImportPath] = true
		path := filepath.Join(pkg.Dir, frontendManifestName)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return result, fmt.Errorf("reading %s: %w", path, err)
		}
		var manifest frontendManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return result, fmt.Errorf("parsing %s: %w", path, err)
		}
		if err := validateFrontendManifest(manifest, pkg.Dir); err != nil {
			return result, fmt.Errorf("validating %s: %w", path, err)
		}
		if seenIDs[manifest.ID] {
			return result, fmt.Errorf("frontend module id %q is contributed more than once", manifest.ID)
		}
		seenIDs[manifest.ID] = true
		contribution := frontendContribution{
			ID: manifest.ID, Framework: manifest.Framework, Package: pkg.ImportPath,
			Root: pkg.Dir, Pages: map[string]string{}, Entries: map[string]string{},
			Styles:           append([]string(nil), manifest.Styles...),
			PeerDependencies: manifest.PeerDependencies,
		}
		for name, relative := range manifest.Pages {
			absolute := filepath.Join(pkg.Dir, filepath.FromSlash(relative))
			contribution.Pages[name] = frontendSourcePath(absolute)
			if prior, ok := result.Pages[name]; ok {
				return result, fmt.Errorf("frontend page %q is contributed by both %s and %s", name, prior, manifest.ID)
			}
			result.Pages[name] = frontendSourcePath(absolute)
		}
		for name, relative := range manifest.Entries {
			contribution.Entries[name] = frontendSourcePath(filepath.Join(pkg.Dir, filepath.FromSlash(relative)))
		}
		contribution.Root = frontendSourcePath(pkg.Dir)
		result.Modules = append(result.Modules, contribution)
	}
	sort.Slice(result.Modules, func(i, j int) bool { return result.Modules[i].ID < result.Modules[j].ID })
	return result, nil
}

func frontendSourcePath(absolute string) string {
	root, err := os.Getwd()
	if err != nil {
		return filepath.ToSlash(absolute)
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return filepath.ToSlash(absolute)
	}
	return filepath.ToSlash(relative)
}

func validateFrontendManifest(manifest frontendManifest, packageDir string) error {
	if manifest.Schema != 1 {
		return fmt.Errorf("unsupported schema %d", manifest.Schema)
	}
	if strings.TrimSpace(manifest.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if manifest.Framework != "react" && manifest.Framework != "vue" {
		return fmt.Errorf("framework must be react or vue")
	}
	for name, relative := range manifest.Pages {
		if err := validateFrontendPath(packageDir, relative); err != nil {
			return fmt.Errorf("page %q: %w", name, err)
		}
	}
	for name, relative := range manifest.Entries {
		if err := validateFrontendPath(packageDir, relative); err != nil {
			return fmt.Errorf("entry %q: %w", name, err)
		}
	}
	for _, relative := range manifest.Styles {
		if err := validateFrontendPath(packageDir, relative); err != nil {
			return fmt.Errorf("style: %w", err)
		}
	}
	return nil
}

func validateFrontendPath(root, relative string) error {
	if relative == "" || filepath.IsAbs(relative) {
		return fmt.Errorf("path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes package directory")
	}
	if _, err := os.Stat(filepath.Join(root, clean)); err != nil {
		return fmt.Errorf("%s: %w", relative, err)
	}
	return nil
}
