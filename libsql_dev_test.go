package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeEnv puts a .env in a temporary directory and makes it the working
// directory, because libsqlDevTarget reads the project it is standing in.
func writeEnv(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	// macOS hands out /var/folders paths that are symlinks to /private/var,
	// and Getwd resolves them, so the expected path has to be resolved too or
	// the comparison fails on the prefix alone.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// `dev` must start a libSQL server only when all four conditions hold, and the
// cases it declines are the point: a hosted database must not get a local
// server racing it, and a server somebody already started must not get a
// second one fighting for the port.
func TestDevStartsALibSQLServerOnlyWhenItShould(t *testing.T) {
	t.Run("another driver is left alone", func(t *testing.T) {
		writeEnv(t, "DB_DRIVER=postgres\nDB_URL=http://127.0.0.1:8080\n")
		addr, _, note, _ := libsqlDevTarget()
		if addr != "" || note != "" {
			t.Errorf("postgres project: addr=%q note=%q, want both empty", addr, note)
		}
	})

	t.Run("a hosted database is left alone, silently", func(t *testing.T) {
		writeEnv(t, "DB_DRIVER=libsql\nDB_URL=https://db-org.turso.io\n")
		addr, _, note, _ := libsqlDevTarget()
		if addr != "" {
			t.Errorf("hosted libSQL got a local server on %q", addr)
		}
		if note != "" {
			t.Errorf("hosted libSQL produced a note (%q); there is nothing to report", note)
		}
	})

	t.Run("a healthy server already listening is left alone", func(t *testing.T) {
		// A real pipeline reply, not a bare socket: a port that accepts and
		// then says nothing is the *unhealthy* case, covered by
		// TestDevTellsAHealthyServerFromABoundPort. Leaving a server alone
		// requires that it can answer.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"baton":null,"results":[
				{"type":"ok","response":{"type":"execute","result":{"cols":[],"rows":[],"affected_row_count":0}}},
				{"type":"ok","response":{"type":"close"}}]}`)
		}))
		defer server.Close()
		writeEnv(t, "DB_DRIVER=libsql\nDB_URL="+server.URL+"\n")

		addr, _, note, fatal := libsqlDevTarget()
		if addr != "" {
			t.Errorf("started a second server on an occupied port %q", addr)
		}
		if !strings.Contains(note, "already listening") {
			t.Errorf("note = %q, want it to say the port is taken", note)
		}
		if fatal {
			t.Error("a healthy server stopped dev")
		}
	})

	t.Run("a missing DB_URL is reported", func(t *testing.T) {
		writeEnv(t, "DB_DRIVER=libsql\n")
		addr, _, note, _ := libsqlDevTarget()
		if addr != "" || !strings.Contains(note, "DB_URL is unset") {
			t.Errorf("addr=%q note=%q", addr, note)
		}
	})

	t.Run("an unreachable loopback port gets a server, at an absolute path", func(t *testing.T) {
		if !HasBinary("sqld") {
			t.Skip("sqld is not installed, and its absence is a note rather than an address")
		}
		// Port 1 needs root to bind, so nothing can be listening on it.
		dir := writeEnv(t, "DB_DRIVER=libsql\nDB_URL=http://127.0.0.1:1\n")
		addr, dbPath, note, _ := libsqlDevTarget()
		if addr != "127.0.0.1:1" {
			t.Errorf("addr = %q, want 127.0.0.1:1 (note: %q)", addr, note)
		}
		if want := filepath.Join(dir, "storage", "data", "libsql"); dbPath != want {
			t.Errorf("dbPath = %q, want %q", dbPath, want)
		}
		if !filepath.IsAbs(dbPath) {
			t.Error("dbPath is relative; sqld resolves it against its own working " +
				"directory, so a relative path makes a second, empty database")
		}
	})

	t.Run("a URL with no port takes libSQL's default", func(t *testing.T) {
		if !HasBinary("sqld") {
			t.Skip("sqld is not installed")
		}
		writeEnv(t, "DB_DRIVER=libsql\nDB_URL=http://127.0.0.1\n")
		addr, _, _, _ := libsqlDevTarget()
		if addr != "" && !strings.HasSuffix(addr, ":8080") {
			t.Errorf("addr = %q, want sqld's default port appended", addr)
		}
	})
}

// envValue reads the project's own .env, and the cases that matter are the
// ones a configuration library would handle and a naive split would not.
func TestEnvValueReadsOneKey(t *testing.T) {
	writeEnv(t, strings.Join([]string{
		"# DB_DRIVER=commented",
		"",
		"DB_DRIVER=libsql",
		`DB_URL="http://127.0.0.1:8080"`,
		"APP_NAME='quoted single'",
		"EQUALS=a=b=c",
		"  SPACED  =  padded  ",
	}, "\n"))

	for key, want := range map[string]string{
		"DB_DRIVER": "libsql",
		"DB_URL":    "http://127.0.0.1:8080",
		"APP_NAME":  "quoted single",
		"EQUALS":    "a=b=c",
		"SPACED":    "padded",
		"ABSENT":    "",
	} {
		if got := envValue(key); got != want {
			t.Errorf("envValue(%q) = %q, want %q", key, got, want)
		}
	}
}

// A libsql project has to say that something must be serving the database,
// because nothing else in the project does.
//
// This is the gap that made `--database libsql` feel broken: pick it, run the
// project, and it panics with `failed to execute SQL:` and an empty message,
// which says nothing about a server not being there. Every other driver
// choice is self-sufficient -- sqlite creates its own file, and mysql and
// postgres at least name a host you would recognise as needing one.
func TestALibSQLProjectSaysWhatHasToBeRunning(t *testing.T) {
	render := func(t *testing.T, driver SQLDriver) string {
		t.Helper()
		dir := t.TempDir()
		cfg := ProjectConfig{
			Name: "says" + string(driver), ModuleName: "example.com/says" + string(driver),
			Preset: PresetMVC, ORM: OrmLemmego, SQLDriver: driver, EnableAuth: true,
		}
		if err := ScaffoldProject(cfg, dir); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, ".env.example"))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}

	libsql := render(t, SQLLibSQL)
	for _, want := range []string{
		"make libsql",    // the command that starts one
		"sqld --db-path", // and the raw form, for anyone not using make
		"is a server",    // why it is needed at all
		"absolute",       // the relative-path trap
	} {
		if !strings.Contains(libsql, want) {
			t.Errorf("a libsql project's .env.example never mentions %q", want)
		}
	}

	// And it stays out of the way of every other choice.
	if sqlite := render(t, SQLSQLite); strings.Contains(sqlite, "make libsql") {
		t.Error("a sqlite project is told to run a libSQL server")
	}
}

// A port being bound is not a working database, and the difference is the
// failure this check exists for.
//
// A sqld whose --db-path was deleted under it -- a project regenerated while
// a server from the previous one was still running -- keeps its HTTP listener
// and fails every statement with `error code 500: unable to open database
// file`. `dev` saw the port occupied, declined to start a working server, and
// the application panicked against the dead one with a message that names no
// server at all.
func TestDevTellsAHealthyServerFromABoundPort(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		wantBad string // substring of the reported reason, "" when healthy
	}{
		{
			name: "a healthy server is left alone",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"baton":null,"results":[
					{"type":"ok","response":{"type":"execute","result":{"cols":[],"rows":[],"affected_row_count":0}}},
					{"type":"ok","response":{"type":"close"}}]}`)
			},
			wantBad: "",
		},
		{
			name: "a sqld whose data directory was removed",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"baton":null,"results":[
					{"type":"error","error":{"message":"error code 500: unable to open database file: /gone/dbs/default/data"}}]}`)
			},
			wantBad: "unable to open database file",
		},
		{
			name: "something that is not libSQL at all",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, "<html>hello</html>")
			},
			wantBad: "not a libSQL pipeline reply",
		},
		{
			name: "a server that refuses the request",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantBad: "HTTP 503",
		},
		{
			// The status code alone names nothing. sqld puts the path that
			// went away in the body, and that is the line worth printing.
			name: "a 500 whose body names the missing file",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = io.WriteString(w, "unable to open database file: /gone/dbs/default/data")
			},
			wantBad: "unable to open database file: /gone",
		},
		{
			name: "a reply with no results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, `{"baton":null,"results":[]}`)
			},
			wantBad: "no result",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			addr := strings.TrimPrefix(server.URL, "http://")

			why := notServing(addr)
			if tc.wantBad == "" {
				if why != "" {
					t.Errorf("a healthy server was reported unhealthy: %q", why)
				}
				return
			}
			if !strings.Contains(why, tc.wantBad) {
				t.Errorf("reason = %q, want it to mention %q", why, tc.wantBad)
			}

			// And the whole point: dev must report this rather than treat the
			// port as taken and walk away.
			writeEnv(t, "DB_DRIVER=libsql\nDB_URL=http://"+addr+"\n")
			gotAddr, _, note, fatal := libsqlDevTarget()
			if gotAddr != "" {
				t.Errorf("started a server on an occupied port %q", gotAddr)
			}
			if strings.Contains(note, "left it alone") {
				t.Errorf("dev walked away from a dead server: %q", note)
			}
			if !strings.Contains(note, "cannot serve a query") {
				t.Errorf("note = %q, want it to say the server cannot serve", note)
			}
			// And it stops dev, so the explanation is the last thing printed
			// rather than the first thing a stack trace buries.
			if !fatal {
				t.Error("dev would have started air against a database it knows is broken")
			}
		})
	}
}
