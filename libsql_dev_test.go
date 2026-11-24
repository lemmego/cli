package cli

import (
	"net"
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
		addr, _, note := libsqlDevTarget()
		if addr != "" || note != "" {
			t.Errorf("postgres project: addr=%q note=%q, want both empty", addr, note)
		}
	})

	t.Run("a hosted database is left alone, silently", func(t *testing.T) {
		writeEnv(t, "DB_DRIVER=libsql\nDB_URL=https://db-org.turso.io\n")
		addr, _, note := libsqlDevTarget()
		if addr != "" {
			t.Errorf("hosted libSQL got a local server on %q", addr)
		}
		if note != "" {
			t.Errorf("hosted libSQL produced a note (%q); there is nothing to report", note)
		}
	})

	t.Run("a server already listening is left alone", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		writeEnv(t, "DB_DRIVER=libsql\nDB_URL=http://"+listener.Addr().String()+"\n")

		addr, _, note := libsqlDevTarget()
		if addr != "" {
			t.Errorf("started a second server on an occupied port %q", addr)
		}
		if !strings.Contains(note, "already listening") {
			t.Errorf("note = %q, want it to say the port is taken", note)
		}
	})

	t.Run("a missing DB_URL is reported", func(t *testing.T) {
		writeEnv(t, "DB_DRIVER=libsql\n")
		addr, _, note := libsqlDevTarget()
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
		addr, dbPath, note := libsqlDevTarget()
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
		addr, _, _ := libsqlDevTarget()
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
