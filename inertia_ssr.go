package cli

import (
	"bytes"
	"fmt"
	"io"
	"net"
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
		fmt.Printf("Inertia SSR server started (PID %d) on http://%s:%d\n", pid, ssrHost, ssrPort)

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

// probeSSR asks the render endpoint to render, and requires a body back.
//
// @inertiajs/core's server also serves /health, but that only reports that the
// process is up — the same thing a port dial reports. Rendering is the
// property worth checking.
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
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the render endpoint answered %d", response.StatusCode)
	}
	if len(bytes.TrimSpace(answer)) == 0 {
		return fmt.Errorf("the render endpoint answered with an empty body, so the SSR bundle is not loaded")
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

func checkPort(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func init() {
	inertiaSSRStartCmd.Flags().IntVar(&ssrPort, "port", 13714, "SSR server port")
	inertiaSSRStartCmd.Flags().StringVar(&ssrHost, "host", "127.0.0.1", "SSR server host")
	inertiaSSRCmd.AddCommand(inertiaSSRStartCmd)
	inertiaSSRCmd.AddCommand(inertiaSSRStopCmd)
	inertiaSSRCmd.AddCommand(inertiaSSRCheckCmd)
}
