package configs

import (
	{{- if ne .SQLDriver "sqlite"}}
	"time"

	{{end}}
	"github.com/lemmego/api/config"
)

func init() {
	config.Set("sql", config.M{
		"default": config.MustEnv("DB_CONNECTION", "{{.SQLDriver.ConnectionName}}"),
		"connections": config.M{
			{{- if eq .SQLDriver "sqlite"}}
			"sqlite": config.M{
				"driver": "sqlite",

				// Deliberately no "url" key. The queue prefers a connection's
				// url over its database, and the url here used to be
				// "file:...?cache=shared&mode=memory" — an in-memory database.
				// The queue therefore ran against a different database from
				// the rest of the application and lost every job on restart.
				"database":                config.MustEnv("DB_DATABASE", "./storage/database.sqlite"),
				"prefix":                  "",
				"foreign_key_constraints": config.MustEnv("DB_FOREIGN_KEYS", true),
			},
			{{- end}}
			{{- if eq .SQLDriver "mysql"}}
			"mysql": config.M{
				"driver":   "mysql",
				"host":     config.MustEnv("DB_HOST", "localhost"),
				"port":     config.MustEnv("DB_PORT", 3306),
				"database": config.MustEnv("DB_DATABASE", "lemmego"),
				"user":     config.MustEnv("DB_USERNAME", "root"),
				"password": config.MustEnv("DB_PASSWORD", ""),

				// The GORM and Bun connectors read this key directly, so its
				// absence used to panic them at boot.
				"options":           config.M{},
				"auto_create":       config.MustEnv("DB_AUTOCREATE", false),
				"max_open_conns":    config.MustEnv("DB_MAX_OPEN_CONNS", 100),
				"max_idle_conns":    config.MustEnv("DB_MAX_IDLE_CONNS", 10),
				"conn_max_lifetime": config.MustEnv("DB_CONN_MAX_LIFETIME", time.Hour*2),
			},
			{{- end}}
			{{- if eq .SQLDriver "postgres"}}
			"pgsql": config.M{
				"driver":   "postgres",
				"host":     config.MustEnv("DB_HOST", "localhost"),
				"port":     config.MustEnv("DB_PORT", 5432),
				"database": config.MustEnv("DB_DATABASE", "lemmego"),
				"user":     config.MustEnv("DB_USERNAME", ""),
				"password": config.MustEnv("DB_PASSWORD", ""),

				// The GORM and Bun connectors read this key directly, so its
				// absence used to panic them at boot.
				//
				// sslmode defaults to disable because lib/pq defaults to
				// require, and a local Postgres almost never serves TLS — so
				// a freshly scaffolded project answered "pq: SSL is not
				// enabled on the server" and could not boot at all. Set
				// DB_SSLMODE to require or verify-full in any deployment that
				// crosses a network; this default is for a database on
				// localhost.
				"options": config.M{
					"sslmode": config.MustEnv("DB_SSLMODE", "disable"),
				},
				"max_open_conns":    config.MustEnv("DB_MAX_OPEN_CONNS", 100),
				"max_idle_conns":    config.MustEnv("DB_MAX_IDLE_CONNS", 10),
				"conn_max_lifetime": config.MustEnv("DB_CONN_MAX_LIFETIME", time.Hour*2),
			},
			{{- end}}
		},
	})
}
