package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/lemmego/mcp/install"
	"github.com/spf13/cobra"
)

var (
	mcpAgents   []string
	mcpNoSkills bool
	mcpNoRules  bool
	mcpNoGuides bool
)

// mcpInstallCmd writes the files an editor needs to find this project's MCP
// server.
//
// A CLI command rather than one reached through `lemmego run`, and that is the
// whole reason this file is here instead of in the application. `run` does `go
// run ./cmd/app`, which boots the application — and a freshly scaffolded
// project whose database credentials are not filled in yet cannot boot. That
// is exactly when somebody wires up their editor.
//
// This repository has already paid for the mistake once: `lemmego new` used to
// generate an app key by running the project's own command through `run`, the
// boot failed on missing credentials, and every new project started life with
// no key at all.
var mcpInstallCmd = &cobra.Command{
	Use:   "mcp:install",
	Short: "Configure AI coding agents to use this project's MCP server",
	Long: `Writes the configuration an AI coding agent needs to work on this project:

  .mcp.json (and the per-editor equivalents)  how to start the MCP server
  CLAUDE.md / AGENTS.md                       always-on guidelines, in a marked block
  .ai/skills/*/SKILL.md                       on-demand skills for the modules in use
  .ai/rules/index.md                          where recorded project rules are indexed

Only the marked block of a guideline file is replaced, so anything you wrote
yourself is preserved. The .ai/rules directory should be committed — those are
your team's decisions; the generated config files can be gitignored.

This does not boot the application, so it works in a project that cannot yet
run. The server itself is started by your editor, as "lemmego run mcp:serve".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		if !isLemmegoProject() {
			return fmt.Errorf("this does not appear to be a Lemmego project directory")
		}

		agents, err := parseAgents(mcpAgents)
		if err != nil {
			return err
		}

		result, err := install.Run(install.Options{
			Dir:        dir,
			Agents:     agents,
			Skills:     !mcpNoSkills,
			Rules:      !mcpNoRules,
			Guidelines: !mcpNoGuides,
		})
		if err != nil {
			return err
		}

		for _, path := range result.Written {
			fmt.Printf("  wrote    %s\n", path)
		}
		for _, path := range result.Skipped {
			fmt.Printf("  current  %s\n", path)
		}
		// The wiring check, and it is not a footnote: writing the
		// configuration and having a server are different things, and only
		// the first is this command's doing. `lemmego new` registers the
		// provider; an existing project has nobody to do it, because Go has
		// no equivalent of Composer's package auto-discovery.
		//
		// Without this the command reported success, the editor started a
		// command that did not exist, and the only signal was a red cross
		// with no reason given.
		if instructions := install.WiringInstructions(result.Wiring); instructions != "" {
			fmt.Println()
			fmt.Println(strings.Repeat("-", 72))
			fmt.Print(instructions)
			fmt.Println(strings.Repeat("-", 72))
			return nil
		}

		fmt.Println()
		fmt.Println("Your editor may need restarting to pick up the MCP server.")
		fmt.Println("Check it is working by asking your agent to list this application's routes.")
		return nil
	},
}

// parseAgents resolves the --agent values, refusing an unknown one rather than
// silently configuring nothing.
func parseAgents(names []string) ([]install.Agent, error) {
	if len(names) == 0 {
		return nil, nil // all of them
	}
	known := make(map[string]install.Agent, len(install.AllAgents()))
	for _, agent := range install.AllAgents() {
		known[string(agent)] = agent
	}

	agents := make([]install.Agent, 0, len(names))
	for _, name := range names {
		agent, ok := known[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			valid := make([]string, 0, len(install.AllAgents()))
			for _, a := range install.AllAgents() {
				valid = append(valid, string(a))
			}
			return nil, fmt.Errorf("unknown agent %q (known: %s)", name, strings.Join(valid, ", "))
		}
		agents = append(agents, agent)
	}
	return agents, nil
}
