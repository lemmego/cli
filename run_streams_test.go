package cli

import (
	"os"
	"strings"
	"testing"
)

// `lemmego run` forwards all three standard streams.
//
// Stdin is the one that was missing, and its absence was invisible for as long
// as nothing read it: a child with no Stdin set gets os.DevNull, so reads
// return EOF immediately. For an ordinary command that looks like "nobody
// typed anything"; for a command that expects input it is an instant, silent
// exit.
//
// It surfaced through `lemmego run mcp:serve`, which speaks JSON-RPC over
// stdin and stdout: the server started, read EOF, treated it as the client
// disconnecting, and exited successfully having written nothing. The editor
// reported CONNECTION_CLOSED, which is true and explains nothing.
func TestRunCommandForwardsEveryStandardStream(t *testing.T) {
	cmd := buildRunCommand([]string{"mcp:serve"})

	if cmd.Stdin != os.Stdin {
		t.Error("stdin is not forwarded, so anything reading it sees EOF at once")
	}
	if cmd.Stdout != os.Stdout {
		t.Error("stdout is not forwarded")
	}
	if cmd.Stderr != os.Stderr {
		t.Error("stderr is not forwarded")
	}
}

// The arguments reach the application unchanged, including a flag that could
// be mistaken for one of the CLI's own.
func TestRunCommandForwardsArgumentsVerbatim(t *testing.T) {
	cmd := buildRunCommand([]string{"mcp:serve", "--allow-writes", "--max-rows", "5"})

	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{"go run ./cmd/app", "mcp:serve", "--allow-writes", "--max-rows 5"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the invocation %q is missing %q", joined, want)
		}
	}
}

// A command that needs no frontend build must not trigger one: `go run` is
// already slow enough that an editor starting an MCP server can time out.
func TestNamespacedCommandsSkipTheFrontendBuild(t *testing.T) {
	if needsFrontend([]string{"mcp:serve"}) {
		t.Error("mcp:serve would trigger a frontend build before starting")
	}
	if needsFrontend([]string{"tasker:work"}) {
		t.Error("tasker:work would trigger a frontend build")
	}
	// Serving HTTP still needs the assets.
	if !needsFrontend(nil) {
		t.Error("running the server should build the frontend")
	}
}

// runApplication surfaces the go tool's failure rather than swallowing it.
//
// The cli package has no ./cmd/app, so `go run` fails immediately — which is
// the point: the caller prints "Error running the app: %v", and if this
// returned nil the user would be told nothing while their command silently
// did not run.
//
// It exercises the real implementation rather than the test seam, so the one
// line that actually shells out is covered by something that asserts a
// behaviour instead of merely visiting it.
func TestRunApplicationReportsAFailureToStart(t *testing.T) {
	err := runApplication([]string{"--help"})
	if err == nil {
		t.Fatal("runApplication() = nil where there is no ./cmd/app to run")
	}
}
