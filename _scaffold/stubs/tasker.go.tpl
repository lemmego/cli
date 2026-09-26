package configs

import (
	"os"
	"time"

	"github.com/lemmego/api/config"
)

func init() {
	settings := config.M{
		"driver": config.MustEnv("TASKER_DRIVER", "{{.QueueDriver}}"),
		{{- if eq .QueueDriver "sql"}}
		"driver_name": config.MustEnv("TASKER_SQL_DRIVER", "{{.SQLDriver.DriverName}}"),
		{{- end}}
		{{- if eq .QueueDriver "redis"}}

		// Defaults to the shared connection; set TASKER_REDIS_ADDR to put the
		// queue somewhere else.
		"redis_addr":     config.MustEnv("TASKER_REDIS_ADDR", redisAddr()),
		"redis_password": config.MustEnv("TASKER_REDIS_PASSWORD", config.MustEnv("REDIS_PASSWORD", "")),
		"redis_db":       config.MustEnv("TASKER_REDIS_DB", 0),
		"redis_prefix":   config.MustEnv("TASKER_REDIS_PREFIX", "tasker:"),
		{{- end}}

		"route_prefix":      config.MustEnv("TASKER_ROUTE_PREFIX", "/tasker"),
		"table_prefix":      config.MustEnv("TASKER_TABLE_PREFIX", "tasker_"),
		"queue":             config.MustEnv("TASKER_QUEUE", "default"),
		"max_attempts":      config.MustEnv("TASKER_MAX_ATTEMPTS", 3),
		"max_open_conns":    config.MustEnv("TASKER_MAX_OPEN_CONNS", 25),
		"max_idle_conns":    config.MustEnv("TASKER_MAX_IDLE_CONNS", 10),
		"conn_max_lifetime": time.Duration(config.MustEnv("TASKER_CONN_MAX_LIFETIME_SEC", 0)) * time.Second,
		"workers": config.M{
			"default": config.MustEnv("TASKER_WORKERS", 3),
		},
		"heartbeat_interval": config.MustEnv("TASKER_HEARTBEAT_SEC", 5),
		"requeue_interval":   config.MustEnv("TASKER_REQUEUE_INTERVAL_SEC", 30),
		"requeue_timeout":    config.MustEnv("TASKER_REQUEUE_SEC", 60),
		"prune_interval":     config.MustEnv("TASKER_PRUNE_INTERVAL_HOURS", 24),
		"prune_after_hours":  config.MustEnv("TASKER_PRUNE_HOURS", 168),
		"autoscale":          config.MustEnv("TASKER_AUTOSCALE", false),
	}

	// Only set when it is actually given. The queue applies every key present
	// here over what it worked out from the application's SQL connection, so
	// an always-present empty dsn would blank that inherited value instead of
	// leaving it alone — and the queue would have no database at all.
	if dsn := os.Getenv("TASKER_DSN"); dsn != "" {
		settings["dsn"] = dsn
	}

	config.Set("tasker", settings)
}
