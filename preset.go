package cli

import (
	"fmt"
)

type ProjectPreset string

const (
	PresetMVC     ProjectPreset = "mvc"
	PresetRESTAPI ProjectPreset = "rest_api"
)

type OrmChoice string

const (
	// OrmLemmego is the framework's own ORM and the default for new
	// projects. GORM and Bun remain available for existing codebases.
	OrmLemmego OrmChoice = "orm"
	OrmGORM    OrmChoice = "gorm"
	OrmBun     OrmChoice = "bun"
)

type FrontendPreset string

const (
	FrontendGoTemplates       FrontendPreset = "go_templates"
	FrontendTempl             FrontendPreset = "templ"
	FrontendInertiaReact      FrontendPreset = "inertia_react"
	FrontendInertiaVue        FrontendPreset = "inertia_vue"
	FrontendTemplInertiaReact FrontendPreset = "templ_inertia_react"
	FrontendTemplInertiaVue   FrontendPreset = "templ_inertia_vue"
)

func (f FrontendPreset) HasInertia() bool {
	return f == FrontendInertiaReact || f == FrontendInertiaVue ||
		f == FrontendTemplInertiaReact || f == FrontendTemplInertiaVue
}

func (f FrontendPreset) HasTempl() bool {
	return f == FrontendTempl || f == FrontendTemplInertiaReact || f == FrontendTemplInertiaVue
}

func (f FrontendPreset) HasNodeDeps() bool {
	return f.HasInertia()
}

const OrmNone OrmChoice = "none"

// ProjectConfig is every decision a new project makes.
//
// The zero value of each field means "use the default", which normalize fills
// in, so a partially populated literal is a valid configuration.
type ProjectConfig struct {
	Name       string
	ModuleName string
	Preset     ProjectPreset
	Frontend   FrontendPreset

	ORM       OrmChoice
	SQLDriver SQLDriver

	CacheDriver    CacheDriver
	QueueDriver    QueueDriver
	SessionDriver  SessionDriver
	FilesystemDisk FilesystemDisk

	EnableAuth bool
	EnableGPA  bool
}

func (c ProjectConfig) HasDatabase() bool { return c.SQLDriver.Enabled() }
func (c ProjectConfig) HasCache() bool    { return c.CacheDriver.Enabled() }
func (c ProjectConfig) HasQueue() bool    { return c.QueueDriver.Enabled() }

// UsesRedis reports whether anything in the project needs a Redis connection,
// and is the only condition under which the shared connection block is written.
//
// This replaced a standalone Redis toggle that was doing three unrelated jobs
// at once — picking the session driver, picking the cache driver, and deciding
// whether the connection block existed. That is why choosing the Redis session
// driver in a project scaffolded without the toggle used to panic on the first
// request: the driver was selected but the connection it reads was never
// written. Deriving the block from the drivers makes that combination
// impossible to express.
func (c ProjectConfig) UsesRedis() bool {
	return c.CacheDriver == CacheRedis ||
		c.QueueDriver == QueueRedis ||
		c.SessionDriver == SessionRedis
}

// One predicate per emission site, so a template never has to nest conditions.
func (c ProjectConfig) UseORMConnector() bool  { return c.HasDatabase() && c.ORM == OrmLemmego }
func (c ProjectConfig) UseGormConnector() bool { return c.HasDatabase() && c.ORM == OrmGORM }
func (c ProjectConfig) UseBunConnector() bool  { return c.HasDatabase() && c.ORM == OrmBun }
func (c ProjectConfig) UseGPA() bool           { return c.HasDatabase() && c.EnableGPA }

// defaultProjectConfig is the single source of defaults.
//
// Both entry points start from it: the interactive form seeds its fields here
// and the flag path fills in whatever was not passed. They used to disagree —
// interactively the default was whichever option happened to be listed first,
// and non-interactively it was a bool flag's zero value — so the same command
// produced different projects depending on how it was invoked.
func defaultProjectConfig(name string) ProjectConfig {
	return ProjectConfig{
		Name:           name,
		Preset:         PresetMVC,
		Frontend:       FrontendGoTemplates,
		ORM:            OrmLemmego,
		SQLDriver:      SQLSQLite,
		CacheDriver:    CacheFile,
		QueueDriver:    QueueSQL,
		SessionDriver:  SessionFile,
		FilesystemDisk: DiskLocal,
		EnableAuth:     true,
		EnableGPA:      false,
	}
}

// normalize fills unset fields from the defaults and resolves what one answer
// implies about another.
func normalize(c *ProjectConfig) {
	d := defaultProjectConfig(c.Name)

	if c.Preset == "" {
		c.Preset = d.Preset
	}
	if c.Frontend == "" {
		c.Frontend = d.Frontend
	}
	if c.SQLDriver == "" {
		c.SQLDriver = d.SQLDriver
	}
	if c.CacheDriver == "" {
		c.CacheDriver = d.CacheDriver
	}
	if c.QueueDriver == "" {
		c.QueueDriver = d.QueueDriver
	}
	if c.SessionDriver == "" {
		c.SessionDriver = d.SessionDriver
	}
	if c.FilesystemDisk == "" {
		c.FilesystemDisk = d.FilesystemDisk
	}
	if c.ORM == "" && c.SQLDriver.Enabled() {
		c.ORM = d.ORM
	}

	// No database means no ORM and no GPA: the connector is what registers the
	// connection, and GPA is registered by the connector.
	if !c.SQLDriver.Enabled() {
		c.ORM = OrmNone
		c.EnableGPA = false
	}

	// A frontend preset is an MVC concept; a REST API has no pages.
	if c.Preset != PresetMVC {
		c.Frontend = FrontendGoTemplates
	}
}

// validate rejects combinations that would scaffold cleanly and then fail at
// runtime.
func validate(c ProjectConfig) error {
	if c.QueueDriver == QueueSQL && !c.HasDatabase() {
		return fmt.Errorf("a SQL queue needs a database: choose a database, or use the redis queue or none")
	}
	if c.EnableAuth && !c.HasDatabase() {
		return fmt.Errorf("authentication needs a database: it scaffolds a users table, a migration and repositories that resolve the connection")
	}
	if c.EnableGPA && !c.HasDatabase() {
		return fmt.Errorf("GPA needs a database: its provider is registered by the SQL connector")
	}
	return nil
}
