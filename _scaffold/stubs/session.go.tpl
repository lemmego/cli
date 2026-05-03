package configs

import (
	"github.com/lemmego/api/config"
	"net/http"
	"time"
)

func init() {
	config.Set("session", config.M{
		"driver": config.MustEnv("SESSION_DRIVER", "{{.SessionDriver}}"),
		"lifetime": config.MustEnv("SESSION_LIFETIME", time.Minute*120),
		"expire_on_close": false,
		"encrypt": false,
		"connection": config.MustEnv("SESSION_CONNECTION", ""),
		"cookie": config.MustEnv("SESSION_COOKIE", "lemmego") + "_session",
		"files": "./storage/session",
		"http_only": config.MustEnv("SESSION_HTTP_ONLY", true),
		"secure":    config.MustEnv("SESSION_SECURE_COOKIE", false),
		"domain":    config.MustEnv("SESSION_DOMAIN", ""),
		"path":      config.MustEnv("SESSION_PATH", "/"),
		"same_site": config.MustEnv("SESSION_SAME_SITE", http.SameSiteLaxMode),
	})
}
