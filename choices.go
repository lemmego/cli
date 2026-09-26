package cli

// The types below name the choices a new project makes. Each is a string so
// templates can compare it directly — {{if eq .CacheDriver "redis"}} — the way
// ProjectPreset and FrontendPreset already do.
//
// Three of them have a None. That is not uniform because the subsystems are
// not: a project can run without a database, a cache or a queue, but not
// without sessions, because the HTTP server wraps its router in the session
// manager and sessions carry the CSRF token, validation errors and flash
// messages that error pages depend on.

// SQLDriver is the database backend.
//
// It is a separate question from OrmChoice: the ORM decides which library maps
// rows to structs, the driver decides what the rows are stored in. Neither
// implies the other.
type SQLDriver string

const (
	SQLNone     SQLDriver = "none"
	SQLSQLite   SQLDriver = "sqlite"
	SQLMySQL    SQLDriver = "mysql"
	SQLPostgres SQLDriver = "postgres"
)

func (d SQLDriver) Enabled() bool { return d != "" && d != SQLNone }

// ConnectionName is the key this driver occupies under sql.connections, which
// is what DB_CONNECTION selects. It is not always the driver name: the
// Postgres connection has always been called "pgsql" while its driver value is
// "postgres".
func (d SQLDriver) ConnectionName() string {
	if d == SQLPostgres {
		return "pgsql"
	}
	return string(d)
}

// DriverName is the connection's "driver" value, and what the migrate command
// reads from DB_DRIVER.
func (d SQLDriver) DriverName() string { return string(d) }

// DefaultPort is the port the server listens on out of the box. SQLite has
// none, being a file.
func (d SQLDriver) DefaultPort() int {
	switch d {
	case SQLMySQL:
		return 3306
	case SQLPostgres:
		return 5432
	}
	return 0
}

// CacheDriver is where cached values are kept.
type CacheDriver string

const (
	// CacheNone leaves the cache out of the project entirely: no dependency,
	// no provider, no config file. It is not the cache module's "null" store,
	// which is registered and configured and simply misses every read.
	CacheNone   CacheDriver = "none"
	CacheFile   CacheDriver = "file"
	CacheMemory CacheDriver = "memory"
	CacheRedis  CacheDriver = "redis"
)

func (d CacheDriver) Enabled() bool { return d != "" && d != CacheNone }

// QueueDriver is where background jobs are kept.
type QueueDriver string

const (
	QueueNone  QueueDriver = "none"
	QueueSQL   QueueDriver = "sql"
	QueueRedis QueueDriver = "redis"
)

func (d QueueDriver) Enabled() bool { return d != "" && d != QueueNone }

// SessionDriver is where session data is kept.
//
// There is no None. The server builds its handler as
// session.LoadAndSave(router), and the session carries the CSRF token,
// validation errors and flash messages — an application without one cannot
// render its own error pages.
type SessionDriver string

const (
	SessionFile   SessionDriver = "file"
	SessionMemory SessionDriver = "memory"
	SessionRedis  SessionDriver = "redis"
)

// FilesystemDisk is the default storage disk.
//
// There is no None, and only these two exist: the resolver understands local
// and s3 and falls back to local for anything else.
type FilesystemDisk string

const (
	DiskLocal FilesystemDisk = "local"
	DiskS3    FilesystemDisk = "s3"
)
