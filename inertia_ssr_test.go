package cli

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

// serveProbe stands a render endpoint up on a real port, because probeSSR
// takes a host and a port rather than a URL.
func serveProbe(t *testing.T, handler http.HandlerFunc) (string, int) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing the test server URL: %v", err)
	}
	host, portText, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("splitting %q: %v", parsed.Host, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parsing port %q: %v", portText, err)
	}
	return host, port
}

// A healthy renderer answers 500 to the probe, and that is the correct answer
// to the question asked: the probe names a component the bundle cannot have,
// so refusing it proves the module graph loaded and the bundle's router ran.
//
// This is the regression. probeSSR used to require 200, so `inertia-ssr check`
// reported every working renderer as unusable — the identical bug
// lemmego/inertia's CheckSSR had and fixed.
func TestProbeAcceptsARefusalWithABody(t *testing.T) {
	host, port := serveProbe(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Page not found: __inertia_probe__"}`))
	})

	if err := probeSSR(host, port); err != nil {
		t.Fatalf("a renderer refusing the probe component is healthy, got: %v", err)
	}
}

// An empty body is the port dial's blind spot: the process is listening and
// the bundle never loaded.
func TestProbeRejectsAnEmptyBody(t *testing.T) {
	host, port := serveProbe(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	err := probeSSR(host, port)
	if err == nil {
		t.Fatal("an empty body means the SSR bundle is not loaded")
	}
	if !strings.Contains(err.Error(), "not loaded") {
		t.Fatalf("the error should name the bundle, got: %v", err)
	}
}

// A 404 is a different fault with a different fix — the renderer is somewhere
// else, or this is a dev server not serving the SSR route — so it must not be
// read as a renderer refusing a component.
func TestProbeRejectsAMissingEndpoint(t *testing.T) {
	host, port := serveProbe(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	err := probeSSR(host, port)
	if err == nil {
		t.Fatal("a 404 means there is no render endpoint there")
	}
	if !strings.Contains(err.Error(), "no render endpoint") {
		t.Fatalf("the error should name the endpoint, got: %v", err)
	}
}

// The probe must reach the port it is given. --port and --host used to sit on
// `start`, where the Node server read neither, so `check` could only ever
// probe the default.
func TestProbeUsesTheGivenPort(t *testing.T) {
	var reached bool
	host, port := serveProbe(t, func(w http.ResponseWriter, r *http.Request) {
		reached = true
		if r.URL.Path != "/render" {
			t.Errorf("probe asked for %q, want /render", r.URL.Path)
		}
		w.Write([]byte(`{"body":"<div></div>"}`))
	})

	if err := probeSSR(host, port); err != nil {
		t.Fatalf("probing the test server: %v", err)
	}
	if !reached {
		t.Fatal("probeSSR did not reach the server it was pointed at")
	}
}

// The flags live on check, which is the command that reads them. On start they
// only changed a printed address, which made the command look configurable
// when it is not.
func TestSSRAddressFlagsAreOnCheckNotStart(t *testing.T) {
	if inertiaSSRCheckCmd.Flags().Lookup("port") == nil ||
		inertiaSSRCheckCmd.Flags().Lookup("host") == nil {
		t.Error("check must accept --port and --host; it is what probes an address")
	}
	if inertiaSSRStartCmd.Flags().Lookup("port") != nil ||
		inertiaSSRStartCmd.Flags().Lookup("host") != nil {
		t.Error("start must not accept --port or --host; the SSR server reads neither")
	}
}
