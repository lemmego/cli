package configs

import (
	"fmt"

	"github.com/lemmego/api/config"
)

// One Redis connection, shared.
//
// Every subsystem that can use Redis reads its own setting first and falls
// back to this, so a project running a single server configures REDIS_* once
// and a project splitting them overrides only what moves.
//
// This file exists exactly when something needs it, which is what stops a
// project selecting a Redis driver while the connection it reads is absent.
func init() {
	config.Set("keyvalue", config.M{
		"connections": config.M{
			"redis": config.M{
				"host":     config.MustEnv("REDIS_HOST", "localhost"),
				"port":     config.MustEnv("REDIS_PORT", 6379),
				"password": config.MustEnv("REDIS_PASSWORD", ""),
			},
		},
	})
}

// redisAddr is the shared host:port the cache and the queue default to. The
// session provider reads the connection settings directly and needs no
// equivalent.
func redisAddr() string {
	return fmt.Sprintf("%s:%d",
		config.MustEnv("REDIS_HOST", "localhost"),
		config.MustEnv("REDIS_PORT", 6379),
	)
}
