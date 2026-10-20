package cli

import (
	"strings"
	"testing"
)

// validate exists to refuse combinations that scaffold cleanly and then fail
// at runtime, so each rule is worth a case: a rule that stops refusing is a
// project that builds and does not boot.
func TestValidateRefusesCombinationsThatCannotWork(t *testing.T) {
	base := func() ProjectConfig {
		return ProjectConfig{
			Name:          "x",
			ModuleName:    "example.com/x",
			Preset:        PresetMVC,
			ORM:           OrmLemmego,
			SQLDriver:     SQLSQLite,
			SessionDriver: SessionFile,
		}
	}

	cases := []struct {
		name   string
		mutate func(*ProjectConfig)
		wants  string
	}{
		{
			// The queue stores its jobs in SQL, so there has to be a database.
			name:   "sql queue without a database",
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLNone; c.QueueDriver = QueueSQL },
			wants:  "SQL queue needs a database",
		},
		{
			// auth scaffolds a users table, a migration and repositories.
			name:   "auth without a database",
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLNone; c.EnableAuth = true },
			wants:  "authentication needs a database",
		},
		{
			name:   "GPA without a database",
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLNone; c.EnableGPA = true },
			wants:  "GPA needs a database",
		},
		{
			// The driver resolves the application's registered connection, so
			// without one it refuses at boot. Refusing here names the fix
			// while the choice is still being made.
			name:   "database sessions without a database",
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLNone; c.SessionDriver = SessionDatabase },
			wants:  "database sessions need a database",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(&cfg)
			err := validate(cfg)
			if err == nil {
				t.Fatalf("validate(%+v) = nil, want a refusal", cfg)
			}
			if !strings.Contains(err.Error(), tc.wants) {
				t.Errorf("validate() = %q, want it to mention %q", err, tc.wants)
			}
		})
	}
}

// And the combinations that do work must not be refused, or the rules are
// worse than none.
func TestValidateAcceptsWorkingCombinations(t *testing.T) {
	working := []ProjectConfig{
		{SQLDriver: SQLSQLite, SessionDriver: SessionDatabase, EnableAuth: true, QueueDriver: QueueSQL},
		{SQLDriver: SQLPostgres, SessionDriver: SessionFile, EnableGPA: true},
		// No database at all is legitimate, as long as nothing needs one.
		{SQLDriver: SQLNone, SessionDriver: SessionMemory, CacheDriver: CacheMemory},
	}
	for _, cfg := range working {
		if err := validate(cfg); err != nil {
			t.Errorf("validate(%+v) = %v, want nil", cfg, err)
		}
	}
}
