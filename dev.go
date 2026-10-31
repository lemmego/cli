package cli

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

type devProcess struct {
	name string
	cmd  *exec.Cmd
}

// ssrBundlePath is where `vite build --ssr` puts the renderer, following
// laravel-vite-plugin's ssrOutputDirectory, which the scaffold keeps.
const ssrBundlePath = "bootstrap/ssr/ssr.js"

// standaloneRendererURL is where that bundle listens: @inertiajs/react's own
// server defaults to 13714 and the generated bundle does not change it.
const standaloneRendererURL = "http://127.0.0.1:13714"

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server with hot reload",
	Run: func(cmd *cobra.Command, args []string) {
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		var processes []devProcess

		// Vite first, and then wait for it, because an Inertia project's
		// server-side renderer is the Vite dev server: @inertiajs/vite serves
		// /__inertia_ssr, and the address is published by writing
		// ./public/hot. Starting air first meant the Go process booted with no
		// hot file, so the first page loads rendered against a standalone
		// renderer that `dev` deliberately never starts — and strict SSR,
		// which is on outside production, turns that into a 500 rather than a
		// quiet degrade.
		//
		// lemmego/inertia re-resolves the address per render now, so this is
		// no longer the difference between working and not. It is the
		// difference between a clean start and a few seconds of errors on the
		// first page somebody opens, which is the first thing they see.
		// The standalone renderer, when there is a built bundle to serve,
		// because that is what keeps server rendering out of the reload loop.
		//
		// Rendering through the Vite dev server means Vite transforms the SSR
		// module graph on demand: measured on a real application, the first
		// render after Vite starts took the better part of a minute, which is
		// long enough that strict SSR reports it as a broken renderer rather
		// than a slow one. This process serves a bundle finished before it
		// started, so there is no warm-up and rebuilding the Go binary does
		// not touch it.
		//
		// The cost, which the printed line says out loud: the bundle is only
		// as current as the last `vite build --ssr`. A component change
		// reaches the browser over HMR and the server-rendered document only
		// after a rebuild.
		var appEnv []string
		if fileExists(ssrBundlePath) {
			processes = append(processes, startDevProcess("ssr", ".", "node", ssrBundlePath))
			appEnv = append(appEnv, "INERTIA_SSR_URL="+standaloneRendererURL)
			fmt.Printf("Server rendering through %s (%s).\n", standaloneRendererURL, ssrBundlePath)
			fmt.Println("  Run `" + npmBinary() + " run build:ssr` after a component change for it to render server-side.")
		}

		if fileExists("package.json") {
			processes = append(processes, startDevProcess("vite", ".", npmBinary(), "run", "dev"))
			waitForViteHotFile()
			if len(appEnv) == 0 && fileExists("resources/js/ssr.tsx") {
				fmt.Println("Server rendering through the Vite dev server, so the first render " +
					"after startup is slow while it transforms the module graph.")
				fmt.Println("  Run `" + npmBinary() + " run build:ssr` once to render through the standalone server instead.")
			}
		}

		// Always: air for Go hot reload
		EnsureBinary("air")
		processes = append(processes, startDevProcessWithEnv("air", ".", appEnv, "air"))

		// If templ files exist: templ generate --watch
		if hasTemplFiles() {
			EnsureBinary("templ")
			processes = append(processes, startDevProcess("templ", ".", "templ", "generate", "--watch", "--proxy", "http://localhost:8080"))
		}

		fmt.Println()
		fmt.Println("Development server started. Press Ctrl+C to stop.")
		fmt.Println()

		waitForShutdown(processes)
	},
}

func startDevProcess(name string, dir string, command string, args ...string) devProcess {
	return startDevProcessWithEnv(name, dir, nil, command, args...)
}

// startDevProcessWithEnv is startDevProcess with extra variables for the
// child, which is how the renderer's address reaches the application: air
// spawns the binary, so anything air inherits the application inherits too.
func startDevProcessWithEnv(name string, dir string, env []string, command string, args ...string) devProcess {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	prefix := processPrefix(name)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	var wg sync.WaitGroup

	drain := func(r io.Reader) {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				for _, line := range strings.Split(strings.TrimRight(string(buf[:n]), "\n"), "\n") {
					if line != "" {
						fmt.Printf("%s %s\n", prefix, line)
					}
				}
			}
			if err != nil {
				return
			}
		}
	}

	wg.Add(2)
	go drain(stdout)
	go drain(stderr)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start %s: %v", name, err)
	}

	go wg.Wait()

	return devProcess{name: name, cmd: cmd}
}

func waitForShutdown(processes []devProcess) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan error, len(processes))
	for _, p := range processes {
		go func(proc devProcess) {
			done <- proc.cmd.Wait()
		}(p)
	}

	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived %s, shutting down...\n", sig)
		killAll(processes)
	case err := <-done:
		if err != nil {
			fmt.Printf("A process exited unexpectedly: %v\n", err)
		}
		killAll(processes)
	}
}

func killAll(processes []devProcess) {
	for _, p := range processes {
		if p.cmd.Process != nil {
			p.cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	for _, p := range processes {
		if p.cmd.Process != nil {
			p.cmd.Process.Wait()
		}
	}
}

func hasTemplFiles() bool {
	found := false
	_ = filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".templ" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func processPrefix(name string) string {
	colors := map[string]string{
		"air":   "\033[36m", // cyan
		"templ": "\033[32m", // green
		"vite":  "\033[35m", // magenta
	}
	reset := "\033[0m"
	c, ok := colors[name]
	if !ok {
		c = "\033[33m" // yellow default
	}
	return fmt.Sprintf("%s[%s]%s", c, name, reset)
}

// viteHotPath mirrors lemmego/inertia's ViteHotPath. It is duplicated rather
// than imported because the CLI must build without the framework in its
// dependency graph.
const viteHotPath = "public/hot"

// viteStartupBudget bounds the wait for the dev server. It is generous
// because a cold Vite with a large dependency graph is slow, and cheap to
// overrun because exceeding it only means proceeding anyway.
const viteStartupBudget = 20 * time.Second

// waitForViteHotFile blocks until Vite publishes its address, or the budget
// expires.
//
// Not fatal on timeout: a project may have a package.json and no Vite, or a
// dev server that fails for reasons of its own, and in both cases the right
// answer is to start the Go server and let its own errors say so rather than
// to refuse to run.
func waitForViteHotFile() {
	// A stale hot file from a previous run would be believed instantly and
	// point at a dev server that is gone. Vite rewrites it on boot, so the
	// wait is for a file newer than this moment.
	started := time.Now()
	if info, err := os.Stat(viteHotPath); err == nil && info.ModTime().Before(started) {
		_ = os.Remove(viteHotPath)
	}

	fmt.Print("> Waiting for the Vite dev server")
	deadline := started.Add(viteStartupBudget)
	for ticks := 0; time.Now().Before(deadline); ticks++ {
		if fileExists(viteHotPath) {
			fmt.Println(" ready")
			return
		}
		if ticks > 0 && ticks%10 == 0 {
			fmt.Print(".")
		}
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println(" not yet; starting the server anyway")
}
