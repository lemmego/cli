package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lemmego/mcp/install"
)

// mcpProject builds the smallest tree mcp:install will accept: a main.go that
// isLemmegoProject recognises, and a go.mod the installer reads to decide which
// skills belong.
func mcpProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "cmd", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	main := "package main\n\nimport _ \"" + lemmegoIndicator + "\"\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "cmd", "app", "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	gomod := "module example.com/app\n\ngo 1.27\n\nrequire (\n" +
		"\tgithub.com/lemmego/api v0.7.2\n" +
		"\tgithub.com/lemmego/migration v0.5.1\n)\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// resetMCPFlags restores the package-level flag variables, which cobra binds
// once and every test would otherwise inherit from the previous one.
func resetMCPFlags(t *testing.T) {
	t.Helper()
	agents, skills, rules, guides := mcpAgents, mcpNoSkills, mcpNoRules, mcpNoGuides
	t.Cleanup(func() {
		mcpAgents, mcpNoSkills, mcpNoRules, mcpNoGuides = agents, skills, rules, guides
	})
	mcpAgents, mcpNoSkills, mcpNoRules, mcpNoGuides = nil, false, false, false
}

func TestMCPInstallWritesTheEditorConfiguration(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
		t.Fatalf("mcp:install error = %v", err)
	}

	for _, path := range []string{
		".mcp.json",
		"CLAUDE.md",
		"AGENTS.md",
		filepath.Join(".ai", "rules", "index.md"),
		filepath.Join(".ai", "skills", "lemmego-migrations", "SKILL.md"),
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Errorf("expected %s: %v", path, err)
		}
	}

	// The server definition is the one thing an editor actually consumes, so
	// it is parsed rather than grepped.
	body, err := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatalf(".mcp.json is not valid JSON: %v\n%s", err, body)
	}
	server, ok := config.MCPServers["lemmego"]
	if !ok {
		t.Fatalf("no lemmego server registered: %s", body)
	}
	if server.Command != "lemmego" || strings.Join(server.Args, " ") != "run mcp:serve" {
		t.Errorf("server = %s %v, want lemmego run mcp:serve", server.Command, server.Args)
	}
}

// Running outside a project must refuse rather than scatter configuration into
// whatever directory the user happened to be in.
func TestMCPInstallRefusesOutsideAProject(t *testing.T) {
	resetMCPFlags(t)
	dir := t.TempDir()
	chdirForTest(t, dir)

	err := mcpInstallCmd.RunE(mcpInstallCmd, nil)
	if err == nil {
		t.Fatal("mcp:install ran outside a Lemmego project")
	}
	if !strings.Contains(err.Error(), "Lemmego project") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("it wrote %d entries into a non-project directory", len(entries))
	}
}

func TestMCPInstallHonoursTheOptOutFlags(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	mcpNoSkills, mcpNoRules, mcpNoGuides = true, true, true
	if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
		t.Fatalf("mcp:install error = %v", err)
	}

	// The MCP config is the one thing that is always written: without it
	// there is no server to find.
	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); err != nil {
		t.Errorf("expected .mcp.json even with every opt-out: %v", err)
	}
	for _, absent := range []string{
		"CLAUDE.md",
		filepath.Join(".ai", "skills"),
		filepath.Join(".ai", "rules"),
	} {
		if _, err := os.Stat(filepath.Join(dir, absent)); !os.IsNotExist(err) {
			t.Errorf("%s was written despite the opt-out", absent)
		}
	}
}

func TestMCPInstallNarrowsToTheNamedAgent(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	mcpAgents = []string{"cursor"}
	if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
		t.Fatalf("mcp:install error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".cursor", "mcp.json")); err != nil {
		t.Errorf("cursor was not configured: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("Claude Code was configured despite --agent cursor")
	}
}

func TestMCPInstallReportsAnUnknownAgent(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	mcpAgents = []string{"emacs"}
	err := mcpInstallCmd.RunE(mcpInstallCmd, nil)
	if err == nil {
		t.Fatal("an unknown agent was accepted")
	}
	// The error has to list the valid names, or the user has to go and read
	// the source to find out what they are.
	for _, want := range []string{"emacs", "claude-code", "cursor"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
}

func TestParseAgentsResolvesEveryKnownName(t *testing.T) {
	for _, agent := range install.AllAgents() {
		got, err := parseAgents([]string{string(agent)})
		if err != nil {
			t.Errorf("parseAgents(%q) error = %v", agent, err)
			continue
		}
		if len(got) != 1 || got[0] != agent {
			t.Errorf("parseAgents(%q) = %v, want [%s]", agent, got, agent)
		}
	}
}

// A name is something a person types, so case and stray spaces must not be a
// failure.
func TestParseAgentsIsForgivingAboutFormatting(t *testing.T) {
	got, err := parseAgents([]string{" Claude-Code ", "CURSOR"})
	if err != nil {
		t.Fatalf("parseAgents() error = %v", err)
	}
	if len(got) != 2 || got[0] != install.ClaudeCode || got[1] != install.Cursor {
		t.Errorf("parseAgents() = %v, want [claude-code cursor]", got)
	}
}

// No names means every agent, which is what makes the flag optional.
func TestParseAgentsWithNoNamesMeansAll(t *testing.T) {
	got, err := parseAgents(nil)
	if err != nil {
		t.Fatalf("parseAgents(nil) error = %v", err)
	}
	if got != nil {
		t.Errorf("parseAgents(nil) = %v, want nil so the installer picks all", got)
	}
}

// Reinstalling is the ordinary case — it is how you pick up a new skill or a
// changed guideline — so it must be idempotent, and it must report what it
// left alone rather than claiming to have rewritten everything.
func TestMCPInstallIsIdempotent(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	// Something hand-written, to prove a reinstall does not eat it.
	claude := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# app\n\nMy own notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
		t.Fatalf("first install error = %v", err)
	}
	first, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}

	if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
		t.Fatalf("second install error = %v", err)
	}
	second, err := os.ReadFile(claude)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Errorf("reinstalling changed CLAUDE.md:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if n := strings.Count(string(second), install.BeginMarker); n != 1 {
		t.Errorf("reinstalling produced %d guideline blocks, want 1", n)
	}
	if !strings.Contains(string(second), "My own notes.") {
		t.Errorf("reinstalling lost the hand-written content:\n%s", second)
	}
}

// The failure this test exists for: mcp:install wrote a correct .mcp.json for
// a project with no server, reported success, and the editor showed a red
// cross with no reason. Configuring an editor and having a server are
// different things.
func TestMCPInstallWarnsWhenTheProjectCannotServe(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t) // go.mod has no lemmego/mcp, and there is no providers.go
	chdirForTest(t, dir)

	stdout := captureStdout(t, func() {
		if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
			t.Fatalf("mcp:install error = %v", err)
		}
	})

	for _, want := range []string{
		"cannot serve MCP yet",
		"go get github.com/lemmego/mcp",
		"&lemmegomcp.Provider{}",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the output does not mention %q:\n%s", want, stdout)
		}
	}
	// The configuration is still written: it is correct, and becomes useful
	// the moment the provider is added.
	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); err != nil {
		t.Errorf("the config was not written: %v", err)
	}
}

// And a wired project must not be nagged.
func TestMCPInstallStaysQuietWhenTheProjectCanServe(t *testing.T) {
	resetMCPFlags(t)
	dir := mcpProject(t)
	chdirForTest(t, dir)

	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		append(gomod, []byte("\nrequire github.com/lemmego/mcp v0.2.0\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "bootstrap"), 0o755); err != nil {
		t.Fatal(err)
	}
	providers := "package bootstrap\n\nimport lemmegomcp \"github.com/lemmego/mcp\"\n\n" +
		"var _ = &lemmegomcp.Provider{}\n"
	if err := os.WriteFile(filepath.Join(dir, "bootstrap", "providers.go"), []byte(providers), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := captureStdout(t, func() {
		if err := mcpInstallCmd.RunE(mcpInstallCmd, nil); err != nil {
			t.Fatalf("mcp:install error = %v", err)
		}
	})

	if strings.Contains(stdout, "cannot serve MCP yet") {
		t.Errorf("a wired project was warned anyway:\n%s", stdout)
	}
	if !strings.Contains(stdout, "restarting") {
		t.Errorf("the ordinary closing message is missing:\n%s", stdout)
	}
}

// captureStdout runs fn with os.Stdout redirected, and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = previous }()

	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, read)
		done <- b.String()
	}()

	fn()
	_ = write.Close()
	return <-done
}
