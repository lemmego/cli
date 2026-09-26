package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// parseChoice resolves a flag value against the values that flag accepts.
//
// The empty string is not valid for any of them, so it unambiguously means
// "the flag was not passed" and falls through to the default.
func parseChoice[T ~string](flag, value string, def T, valid ...T) (T, error) {
	if value == "" {
		return def, nil
	}
	for _, candidate := range valid {
		if T(value) == candidate {
			return candidate, nil
		}
	}

	names := make([]string, len(valid))
	for i, candidate := range valid {
		names[i] = string(candidate)
	}
	var zero T
	return zero, fmt.Errorf("invalid --%s %q: use one of %s", flag, value, strings.Join(names, ", "))
}

func collectNonInteractiveProjectConfig(dirname string) (*ProjectConfig, error) {
	if projectFlags.Module == "" {
		return nil, fmt.Errorf("--module is required with --non-interactive")
	}

	d := defaultProjectConfig(dirname)
	cfg := d
	cfg.ModuleName = projectFlags.Module

	// --redis is shorthand for "use Redis wherever I did not say otherwise",
	// so it changes the defaults rather than the answers.
	cacheDefault, queueDefault, sessionDefault := d.CacheDriver, d.QueueDriver, d.SessionDriver
	if projectFlags.Redis {
		cacheDefault, queueDefault, sessionDefault = CacheRedis, QueueRedis, SessionRedis
	}

	var err error
	if cfg.Preset, err = parseChoice("preset", projectFlags.Preset, d.Preset,
		PresetMVC, PresetRESTAPI); err != nil {
		return nil, err
	}
	if cfg.Frontend, err = parseChoice("frontend", projectFlags.Frontend, d.Frontend,
		FrontendGoTemplates, FrontendTempl, FrontendInertiaReact, FrontendInertiaVue,
		FrontendTemplInertiaReact, FrontendTemplInertiaVue); err != nil {
		return nil, err
	}
	if cfg.SQLDriver, err = parseChoice("database", projectFlags.Database, d.SQLDriver,
		SQLSQLite, SQLMySQL, SQLPostgres, SQLNone); err != nil {
		return nil, err
	}
	if cfg.ORM, err = parseChoice("orm", projectFlags.ORM, d.ORM,
		OrmLemmego, OrmGORM, OrmBun, OrmNone); err != nil {
		return nil, err
	}
	if cfg.CacheDriver, err = parseChoice("cache", projectFlags.Cache, cacheDefault,
		CacheFile, CacheMemory, CacheRedis, CacheNone); err != nil {
		return nil, err
	}
	if cfg.QueueDriver, err = parseChoice("queue", projectFlags.Queue, queueDefault,
		QueueSQL, QueueRedis, QueueNone); err != nil {
		return nil, err
	}
	if cfg.SessionDriver, err = parseChoice("session", projectFlags.Session, sessionDefault,
		SessionFile, SessionMemory, SessionRedis); err != nil {
		return nil, err
	}
	if cfg.FilesystemDisk, err = parseChoice("disk", projectFlags.Disk, d.FilesystemDisk,
		DiskLocal, DiskS3); err != nil {
		return nil, err
	}

	cfg.EnableAuth = projectFlags.Auth
	cfg.EnableGPA = projectFlags.GPA

	normalize(&cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// setupMode is the first question: how much does this user want to be asked?
type setupMode string

const (
	modeQuick  setupMode = "quick"
	modeCustom setupMode = "custom"
)

// collectProjectConfig asks the questions.
//
// It is one form rather than several, so the user can page backwards through
// every answer. Groups that do not apply are hidden rather than skipped by
// running a second form, which is what the previous two-form arrangement was
// working around.
//
// Every field is seeded from defaultProjectConfig, and huh preselects the
// option matching the bound value — so what the form offers first and what a
// flag defaults to are the same value by construction.
func collectProjectConfig(dirname string, enableExperimental bool) *ProjectConfig {
	cfg := defaultProjectConfig(dirname)
	mode := modeQuick

	custom := func() bool { return mode != modeCustom }

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module name").
				Description("For example github.com/username/repo").
				Value(&cfg.ModuleName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("module name is required")
					}
					return nil
				}),
			huh.NewSelect[setupMode]().
				Title("How would you like to set this up?").
				Options(
					huh.NewOption("Quick start — SQLite, file cache, SQL queue, file sessions", modeQuick),
					huh.NewOption("Customise — choose a driver for each part", modeCustom),
				).
				Value(&mode),
		),

		// Asked either way: nobody can pick these for you.
		huh.NewGroup(
			huh.NewSelect[ProjectPreset]().
				Title("Preset").
				Options(
					huh.NewOption("MVC", PresetMVC),
					huh.NewOption("REST API", PresetRESTAPI),
				).
				Value(&cfg.Preset),
			huh.NewConfirm().
				Title("Scaffold authentication?").
				Description("Adds a users table, register and login routes, a model and repositories.").
				Value(&cfg.EnableAuth),
		),

		huh.NewGroup(
			huh.NewSelect[FrontendPreset]().
				Title("Frontend").
				Options(
					huh.NewOption("Go Templates", FrontendGoTemplates),
					huh.NewOption("Templ", FrontendTempl),
					huh.NewOption("Inertia (React)", FrontendInertiaReact),
					huh.NewOption("Inertia (Vue)", FrontendInertiaVue),
					huh.NewOption("Templ + Inertia (React)", FrontendTemplInertiaReact),
					huh.NewOption("Templ + Inertia (Vue)", FrontendTemplInertiaVue),
				).
				Value(&cfg.Frontend),
		).WithHideFunc(func() bool { return cfg.Preset != PresetMVC }),

		huh.NewGroup(
			huh.NewSelect[SQLDriver]().
				Title("Database").
				Options(
					huh.NewOption("SQLite — a file under ./storage, nothing to install", SQLSQLite),
					huh.NewOption("PostgreSQL", SQLPostgres),
					huh.NewOption("MySQL / MariaDB", SQLMySQL),
					huh.NewOption("None", SQLNone),
				).
				Value(&cfg.SQLDriver),
		).WithHideFunc(custom),

		huh.NewGroup(
			huh.NewSelect[OrmChoice]().
				Title("SQL layer").
				Options(
					huh.NewOption("Lemmego ORM", OrmLemmego),
					huh.NewOption("GORM", OrmGORM),
					huh.NewOption("Bun", OrmBun),
				).
				Value(&cfg.ORM),
		).WithHideFunc(func() bool { return custom() || !cfg.SQLDriver.Enabled() }),

		huh.NewGroup(
			huh.NewSelect[CacheDriver]().
				Title("Cache").
				Description("File is shared with the CLI; memory is private to the server process, so cache:clear cannot reach it.").
				Options(
					huh.NewOption("File", CacheFile),
					huh.NewOption("Memory", CacheMemory),
					huh.NewOption("Redis", CacheRedis),
					huh.NewOption("None", CacheNone),
				).
				Value(&cfg.CacheDriver),
		).WithHideFunc(custom),

		huh.NewGroup(
			huh.NewSelect[QueueDriver]().
				Title("Background queue").
				// The SQL option is withdrawn rather than offered and then
				// rejected, so the form cannot produce a combination validate
				// would turn down.
				OptionsFunc(func() []huh.Option[QueueDriver] {
					options := []huh.Option[QueueDriver]{}
					if cfg.SQLDriver.Enabled() {
						options = append(options,
							huh.NewOption("SQL — shares the application database", QueueSQL))
					}
					return append(options,
						huh.NewOption("Redis", QueueRedis),
						huh.NewOption("None", QueueNone),
					)
				}, &cfg.SQLDriver).
				Value(&cfg.QueueDriver),
		).WithHideFunc(custom),

		huh.NewGroup(
			huh.NewSelect[SessionDriver]().
				Title("Sessions").
				Description("Required: sessions carry the CSRF token, validation errors and flash messages.").
				Options(
					huh.NewOption("File", SessionFile),
					huh.NewOption("Memory — lost on restart", SessionMemory),
					huh.NewOption("Redis", SessionRedis),
				).
				Value(&cfg.SessionDriver),
			huh.NewSelect[FilesystemDisk]().
				Title("Default storage disk").
				Options(
					huh.NewOption("Local", DiskLocal),
					huh.NewOption("Amazon S3", DiskS3),
				).
				Value(&cfg.FilesystemDisk),
		).WithHideFunc(custom),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable GPA? (experimental)").
				Value(&cfg.EnableGPA),
		).WithHideFunc(func() bool {
			return custom() || !enableExperimental || !cfg.SQLDriver.Enabled()
		}),
	)

	if err := form.Run(); err != nil {
		return nil
	}

	normalize(&cfg)
	if err := validate(cfg); err != nil {
		fmt.Println("That combination will not work:", err)
		return nil
	}
	return &cfg
}
