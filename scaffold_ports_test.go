package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The application's port must not collide with any service's port in the same
// generated .env.
//
// This test exists because it did. libsql arrived with DB_URL defaulting to
// http://127.0.0.1:8080 — sqld's own default — while APP_PORT had been 8080
// since the beginning, so `lemmego new --database libsql` produced a project
// whose application could not bind its own port. Nothing caught it: the
// project compiles, and the clash only shows when you run it.
//
// The application now listens on 1490, which belongs to nothing in
// /etc/services and is above the privileged range.
func TestTheAppPortDoesNotCollideWithAnyServicePort(t *testing.T) {
	portOf := regexp.MustCompile(`:(\d{2,5})\b`)

	for _, driver := range []SQLDriver{SQLSQLite, SQLMySQL, SQLPostgres, SQLLibSQL} {
		t.Run(string(driver), func(t *testing.T) {
			dir := t.TempDir()
			cfg := ProjectConfig{
				Name:       "ports" + string(driver),
				ModuleName: "example.com/ports" + string(driver),
				Preset:     PresetMVC,
				ORM:        OrmLemmego,
				SQLDriver:  driver,
				EnableAuth: true,
			}
			if err := ScaffoldProject(cfg, dir); err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			body, err := os.ReadFile(filepath.Join(dir, ".env.example"))
			if err != nil {
				t.Fatal(err)
			}

			var appPort string
			services := map[string]string{} // variable -> port
			for _, line := range strings.Split(string(body), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
					continue
				}
				name, value, _ := strings.Cut(line, "=")
				switch name {
				case "APP_PORT":
					appPort = strings.TrimSpace(value)
				case "APP_URL":
					// Carries the same port by construction; checked below.
				default:
					if m := portOf.FindStringSubmatch(value); m != nil {
						services[name] = m[1]
					}
				}
			}

			if appPort == "" {
				t.Fatal("APP_PORT is not set in .env.example")
			}
			for name, port := range services {
				if port == appPort {
					t.Errorf("%s uses port %s, which is APP_PORT — the application cannot bind it",
						name, port)
				}
			}

			// APP_URL must agree with APP_PORT, or one of them is a lie.
			if !strings.Contains(string(body), "APP_URL=http://localhost:"+appPort) {
				t.Errorf("APP_URL does not carry APP_PORT (%s)", appPort)
			}
		})
	}
}

// Every APP_URL fallback written in Go must agree with the .env default, or a
// module computes links against a port the application is not listening on.
func TestEveryAppURLFallbackAgreesWithTheEnvDefault(t *testing.T) {
	wanted := "http://localhost:1490"
	var offenders []string
	err := filepath.Walk("_scaffold", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(line, `"APP_URL"`) && !strings.Contains(line, wanted) {
				offenders = append(offenders, path+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range offenders {
		t.Errorf("APP_URL fallback disagrees with %s\n  %s", wanted, o)
	}
}
