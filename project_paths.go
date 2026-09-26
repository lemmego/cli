package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// projectPaths is where this project puts generated code.
//
// The values come from the project itself rather than from constants here. A
// project's paths live in internal/configs, which is Go source with an init
// function — only the project's own binary can evaluate it, so the CLI asks
// instead of guessing. That is what makes MODEL_PATH, or an edited app.go,
// actually move where a generated model lands.
type projectPaths struct {
	ConfigPath     string `json:"config_path"`
	CommandPath    string `json:"command_path"`
	HandlerPath    string `json:"handler_path"`
	InputPath      string `json:"input_path"`
	MiddlewarePath string `json:"middleware_path"`
	MigrationPath  string `json:"migration_path"`
	ModelPath      string `json:"model_path"`
	RoutePath      string `json:"route_path"`
}

// conventionalPaths is what a project gets when it cannot be asked — outside a
// project, or when it does not build. These match the directories the scaffold
// creates.
func conventionalPaths() projectPaths {
	return projectPaths{
		ConfigPath:     "./internal/configs",
		CommandPath:    "./internal/commands",
		HandlerPath:    "./internal/handlers",
		InputPath:      "./internal/inputs",
		MiddlewarePath: "./internal/middleware",
		MigrationPath:  "./internal/migrations",
		ModelPath:      "./internal/models",
		RoutePath:      "./internal/routes",
	}
}

var (
	pathsOnce   sync.Once
	pathsCached projectPaths
)

// Paths returns where generated code goes, asking the project once.
func Paths() projectPaths {
	pathsOnce.Do(func() { pathsCached = resolveProjectPaths() })
	return pathsCached
}

// pathsTimeout bounds the build the question costs. Asking compiles the
// project, so it is seconds rather than milliseconds — acceptable for a
// command that is about to write a file and think about it, and the answer is
// cached for the rest of the run.
const pathsTimeout = 90 * time.Second

func resolveProjectPaths() projectPaths {
	fallback := conventionalPaths()
	if !isLemmegoProject() {
		return fallback
	}

	command := exec.Command("go", "run", "./cmd/app", "config:paths")
	var stdout bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = nil

	done := make(chan error, 1)
	if err := command.Start(); err != nil {
		return fallback
	}
	go func() { done <- command.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			warnPathsUnavailable(err)
			return fallback
		}
	case <-time.After(pathsTimeout):
		_ = command.Process.Kill()
		warnPathsUnavailable(fmt.Errorf("timed out after %s", pathsTimeout))
		return fallback
	}

	// The project prints one JSON line, but anything it logged during boot
	// comes first, so take the last line rather than the whole buffer.
	var paths projectPaths
	if err := json.Unmarshal(lastLine(stdout.Bytes()), &paths); err != nil {
		warnPathsUnavailable(err)
		return fallback
	}
	return merge(paths, fallback)
}

// warnPathsUnavailable says why the conventional paths are being used. Silence
// would be worse: a project that configured ./domain/models and sees a file in
// ./internal/models has no way to tell whether the setting was ignored or the
// question failed.
func warnPathsUnavailable(err error) {
	fmt.Fprintf(os.Stderr,
		"> Could not ask the project where generated code goes (%v); using the conventional paths.\n", err)
}

func lastLine(output []byte) []byte {
	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	return lines[len(lines)-1]
}

// merge fills anything the project left empty, so a partial answer still
// produces a usable set.
func merge(paths, fallback projectPaths) projectPaths {
	if paths.ConfigPath == "" {
		paths.ConfigPath = fallback.ConfigPath
	}
	if paths.CommandPath == "" {
		paths.CommandPath = fallback.CommandPath
	}
	if paths.HandlerPath == "" {
		paths.HandlerPath = fallback.HandlerPath
	}
	if paths.InputPath == "" {
		paths.InputPath = fallback.InputPath
	}
	if paths.MiddlewarePath == "" {
		paths.MiddlewarePath = fallback.MiddlewarePath
	}
	if paths.MigrationPath == "" {
		paths.MigrationPath = fallback.MigrationPath
	}
	if paths.ModelPath == "" {
		paths.ModelPath = fallback.ModelPath
	}
	if paths.RoutePath == "" {
		paths.RoutePath = fallback.RoutePath
	}
	return paths
}
