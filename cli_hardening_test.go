package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateDestinationRejectsUnsafePaths(t *testing.T) {
	for _, dirname := range []string{"", ".", "..", "../outside", "/tmp/project"} {
		if _, err := validateDestination(dirname); err == nil {
			t.Errorf("expected %q to be rejected", dirname)
		}
	}
}

func TestValidateDestinationContentsDetectsHiddenFiles(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "project")
	if err := os.Mkdir(destination, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, ".gitignore"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := validateDestinationContents(destination); err == nil {
		t.Fatal("expected hidden file to make destination non-empty")
	}
}

func TestValidateDestinationRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "project")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if _, err := validateDestination("project"); err == nil {
		t.Fatal("expected symlink destination to be rejected")
	}
}

func TestFetchLatestScaffoldPreservesCacheOnHTTPFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	cacheDir := filepath.Join(root, scaffoldCacheDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "_scaffold"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "VERSION"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(cacheDir, "_scaffold", "old.txt")
	if err := os.WriteFile(oldFile, []byte("old cache"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	restoreScaffoldHTTP(t, server.URL, &http.Client{Timeout: time.Second})

	if fetchLatestScaffold() {
		t.Fatal("expected failed fetch to return false")
	}
	assertCacheFile(t, filepath.Join(cacheDir, "VERSION"), "old")
	assertCacheFile(t, oldFile, "old cache")
}

func TestFetchLatestScaffoldUsesTimeoutAndPreservesCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	cacheDir := filepath.Join(root, scaffoldCacheDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "_scaffold"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "VERSION"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()
	restoreScaffoldHTTP(t, server.URL, &http.Client{Timeout: 10 * time.Millisecond})

	if fetchLatestScaffold() {
		t.Fatal("expected timed out fetch to return false")
	}
	assertCacheFile(t, filepath.Join(cacheDir, "VERSION"), "old")
}

func TestFetchLatestScaffoldAtomicallyReplacesCache(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	cacheDir := filepath.Join(root, scaffoldCacheDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "_scaffold"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "VERSION"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	archive := scaffoldArchive(t, "new content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			_, _ = io.WriteString(w, "new")
			return
		}
		w.Write(archive)
	}))
	defer server.Close()
	oldVersionURL, oldTarballURL, oldClient := scaffoldVersionURL, scaffoldTarballURL, scaffoldHTTPClient
	defer func() {
		scaffoldVersionURL, scaffoldTarballURL, scaffoldHTTPClient = oldVersionURL, oldTarballURL, oldClient
	}()
	scaffoldVersionURL = server.URL + "/version"
	scaffoldTarballURL = server.URL + "/archive"
	scaffoldHTTPClient = &http.Client{Timeout: time.Second}

	if !fetchLatestScaffold() {
		t.Fatal("expected successful fetch")
	}
	assertCacheFile(t, filepath.Join(cacheDir, "VERSION"), "new")
	assertCacheFile(t, filepath.Join(cacheDir, "_scaffold", "new.txt"), "new content")
}

func restoreScaffoldHTTP(t *testing.T, url string, client *http.Client) {
	t.Helper()
	oldVersionURL, oldTarballURL, oldClient := scaffoldVersionURL, scaffoldTarballURL, scaffoldHTTPClient
	t.Cleanup(func() {
		scaffoldVersionURL, scaffoldTarballURL, scaffoldHTTPClient = oldVersionURL, oldTarballURL, oldClient
	})
	scaffoldVersionURL = url
	scaffoldTarballURL = url
	scaffoldHTTPClient = client
}

func scaffoldArchive(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	data := []byte(content)
	if err := tw.WriteHeader(&tar.Header{Name: "repo/_scaffold/new.txt", Mode: 0644, Size: int64(len(data))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func assertCacheFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
