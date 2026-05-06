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

	"github.com/spf13/cobra"
)

type devProcess struct {
	name string
	cmd  *exec.Cmd
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server with hot reload",
	Run: func(cmd *cobra.Command, args []string) {
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		var processes []devProcess

		// Always: air for Go hot reload
		EnsureBinary("air")
		processes = append(processes, startDevProcess("air", ".", "air"))

		// If templ files exist: templ generate --watch
		if hasTemplFiles() {
			EnsureBinary("templ")
			processes = append(processes, startDevProcess("templ", ".", "templ", "generate", "--watch", "--proxy", "http://localhost:8080"))
		}

		// If node deps exist: vite dev server
		if fileExists("package.json") {
			processes = append(processes, startDevProcess("vite", ".", npmBinary(), "run", "dev"))
		}

		fmt.Println()
		fmt.Println("Development server started. Press Ctrl+C to stop.")
		fmt.Println()

		waitForShutdown(processes)
	},
}

func startDevProcess(name string, dir string, command string, args ...string) devProcess {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

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
	matches, _ := filepath.Glob("**/*.templ")
	if len(matches) > 0 {
		return true
	}
	matches, _ = filepath.Glob("*/*.templ")
	return len(matches) > 0
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
