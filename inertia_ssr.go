package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const ssrPidFile = ".ssr.pid"

// Where `check` looks for the renderer.
//
// They are flags on `check` alone. `start` used to carry them too, which was
// a lie in both directions: @inertiajs/core's server reads neither — its port
// is fixed when the bundle is built, by the vite plugin's
// inertia({ssr: {port}}) option — so passing --port only changed the address
// start printed, not the one it listened on.
var ssrPort int
var ssrHost string

var inertiaSSRCmd = &cobra.Command{
	Use:   "inertia-ssr",
	Short: "Manage the Inertia SSR server",
}

var inertiaSSRStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Inertia SSR server",
	Run: func(cmd *cobra.Command, args []string) {
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		ssrPath := filepath.Join("bootstrap", "ssr", "ssr.js")
		if !fileExists(ssrPath) {
			fmt.Println("Error: SSR server script not found at bootstrap/ssr/ssr.js")
			return
		}

		// Check if already running
		if pid, err := readPidFile(); err == nil {
			if processRunning(pid) {
				fmt.Printf("SSR server is already running (PID %d)\n", pid)
				return
			}
			os.Remove(ssrPidFile)
		}

		// No --port or --host: @inertiajs/core's SSR server reads neither.
		// Its port is fixed when the bundle is built, by the vite plugin's
		// inertia({ssr: {port}}) option, so passing them here only made the
		// command look configurable.
		nodeCmd := exec.Command("node", ssrPath)
		nodeCmd.Stdout = os.Stdout
		nodeCmd.Stderr = os.Stderr

		if err := nodeCmd.Start(); err != nil {
			fmt.Printf("Error starting SSR server: %v\n", err)
			return
		}

		pid := nodeCmd.Process.Pid
		os.WriteFile(ssrPidFile, []byte(strconv.Itoa(pid)), 0644)
		// No address, for the same reason there is no --port: this command
		// does not choose one and cannot read the one the bundle baked in, so
		// naming a port here would be a guess printed as a fact. `check` is
		// what answers which address is actually serving.
		fmt.Printf("Inertia SSR server started (PID %d)\n", pid)

		// Wait for signal
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig

		nodeCmd.Process.Signal(syscall.SIGTERM)
		os.Remove(ssrPidFile)
		fmt.Println("\nSSR server stopped")
	},
}

var inertiaSSRStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Inertia SSR server",
	Run: func(cmd *cobra.Command, args []string) {
		pid, err := readPidFile()
		if err != nil {
			fmt.Println("SSR server is not running")
			return
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			os.Remove(ssrPidFile)
			fmt.Println("SSR server is not running")
			return
		}

		proc.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)

		if !processRunning(pid) {
			os.Remove(ssrPidFile)
			fmt.Println("SSR server stopped")
		} else {
			proc.Signal(syscall.SIGKILL)
			os.Remove(ssrPidFile)
			fmt.Println("SSR server killed")
		}
	},
}

var inertiaSSRCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check that the Inertia SSR server can actually render",
	Run: func(cmd *cobra.Command, args []string) {
		// The process being alive is not the question. A Node process
		// listening with a broken module graph passes a liveness check and
		// fails every render — and because Inertia falls back to
		// client-rendered HTML, the site answers 200 throughout. So this asks
		// the server to render, which is the only thing that distinguishes
		// working from listening.
		if pid, err := readPidFile(); err == nil && !processRunning(pid) {
			os.Remove(ssrPidFile)
			fmt.Println("SSR server is not running")
			os.Exit(1)
		}

		if err := probeSSR(ssrHost, ssrPort); err != nil {
			fmt.Printf("SSR server is not rendering: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("SSR server is rendering on http://%s:%d\n", ssrHost, ssrPort)
	},
}

// probeSSR asks the render endpoint to render, and reads the answer.
//
// @inertiajs/core's server also serves /health, but that only reports that the
// process is up — the same thing a port dial reports. Rendering is the
// property worth checking.
//
// The component asked for is deliberately one the bundle cannot have, so the
// renderer is *expected* to refuse it. The refusal is the signal: the process
// is up, the module graph loaded, and the bundle's router ran far enough to
// decide the component does not exist. **The status is therefore not required
// to be 200** — a correct renderer answering this question says 500.
//
// Requiring 200 is what this used to do, and it meant `check` reported every
// healthy renderer as unusable. lemmego/inertia's CheckSSR had the identical
// bug and fixed it; the logic is duplicated rather than imported because the
// CLI deliberately keeps the framework out of its dependency graph.
//
// That makes this a shallow check: it cannot tell a renderer that fails only
// on unknown components from one that fails on all of them. Proving a renderer
// produces documents needs a props-free page to ask for, which is the
// application's to supply — lemmego/inertia's WithSSRProbeComponent is where
// that lives.
func probeSSR(host string, port int) error {
	body := []byte(`{"component":"__inertia_probe__","props":{},"url":"/","version":"","encryptHistory":false,"clearHistory":false}`)
	endpoint := fmt.Sprintf("http://%s:%d/render", host, port)

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	answer, err := io.ReadAll(io.LimitReader(response.Body, 1<<16))
	if err != nil {
		return err
	}

	// An empty body means the bundle never loaded. This is the port dial's
	// blind spot and the reason this function exists.
	if len(bytes.TrimSpace(answer)) == 0 {
		return fmt.Errorf("the render endpoint answered with an empty body, so the SSR bundle is not loaded")
	}

	// A 404 is the endpoint itself being wrong — a renderer at another path,
	// or a dev server not serving the SSR route — rather than a renderer
	// refusing a component. Worth separating, because the fix is different:
	// configuration rather than a broken bundle.
	if response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("there is no render endpoint at %s", endpoint)
	}

	return nil
}

func readPidFile() (int, error) {
	data, err := os.ReadFile(ssrPidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func processRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	return proc.Signal(syscall.Signal(0)) == nil
}

func init() {
	inertiaSSRCheckCmd.Flags().IntVar(&ssrPort, "port", 13714, "Port the SSR server listens on")
	inertiaSSRCheckCmd.Flags().StringVar(&ssrHost, "host", "127.0.0.1", "Host the SSR server listens on")
	inertiaSSRCmd.AddCommand(inertiaSSRStartCmd)
	inertiaSSRCmd.AddCommand(inertiaSSRStopCmd)
	inertiaSSRCmd.AddCommand(inertiaSSRCheckCmd)
}
