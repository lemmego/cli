package plugins

import (
	"database/sql"

	"github.com/lemmego/observe"
{{- if eq .SQLDriver "sqlite"}}
	sqlite "github.com/glebarez/go-sqlite"
{{- else if eq .SQLDriver "mysql"}}
	"github.com/go-sql-driver/mysql"
{{- else if eq .SQLDriver "postgres"}}
	"github.com/lib/pq"
{{- else if eq .SQLDriver "libsql"}}
	"github.com/tursodatabase/libsql-client-go/libsql"
{{- end}}
)

// ObservedDriver is the registered name of this application's instrumented
// database/sql driver.
//
// **This is what makes observe show a request's queries**, and without it the
// dashboard shows requests with nothing underneath them — which reads as "this
// application makes no queries" rather than as a missing wrapper.
//
// Collected at the driver layer rather than through an ORM hook, and that is
// the whole design: one wrapper covers the ORM, raw database/sql, the
// migrator, the queue's job store and the session store. An ORM-level hook
// would cover one of those.
//
// The dialect still comes from DB_DRIVER, so `lemmego run migrate` and the
// ORM generate the same SQL as before — this changes who opens the
// connection, not what is sent over it.
const ObservedDriver = "{{.SQLDriver}}-observed"

func init() {
	sql.Register(ObservedDriver, observe.Wrap(
{{- if eq .SQLDriver "sqlite"}}
		&sqlite.Driver{},
{{- else if eq .SQLDriver "mysql"}}
		&mysql.MySQLDriver{},
{{- else if eq .SQLDriver "postgres"}}
		&pq.Driver{},
{{- else if eq .SQLDriver "libsql"}}
		libsql.Driver{},
{{- end}}
	))
}
