package configs

// Kept in step with cache.ConfigStub in github.com/lemmego/cache, so a project
// scaffolded by the CLI and one that ran `lemmego publish --tags=config` get
// the same file.

import "github.com/lemmego/api/config"

func init() {
	config.Set("cache", config.M{
		// file keeps the cache on disk, so the CLI and the server share it.
		// memory is faster but private to one process, which leaves
		// `lemmego run cache:clear` unable to reach the server's cache.
		"driver": config.MustEnv("CACHE_DRIVER", "{{.CacheDriver}}"),

		// Every key is written under this prefix, and it is what bounds a
		// flush — which matters on a Redis database shared with sessions.
		"prefix": config.MustEnv("CACHE_PREFIX", "lemmego_cache:"),

		// Seconds.
		"ttl": config.MustEnv("CACHE_TTL", 3600),

		// json is portable and readable; gob preserves Go types exactly.
		"codec": config.MustEnv("CACHE_CODEC", "json"),

		// Publish cache events onto the application emitter. Listeners run
		// synchronously on the goroutine that touched the cache.
		"events": config.MustEnv("CACHE_EVENTS", false),

		// Serve without a cache rather than failing to boot when the backend
		// is unreachable. Every read then misses.
		"lenient": config.MustEnv("CACHE_LENIENT", false),

		"stores": config.M{
			{{- if eq .CacheDriver "file"}}
			"file": config.M{
				"path": config.MustEnv("CACHE_FILE_PATH", "storage/framework/cache"),
			},
			{{- end}}
			{{- if eq .CacheDriver "redis"}}
			// Defaults to the shared connection; set CACHE_REDIS_ADDR to put
			// the cache somewhere else.
			"redis": config.M{
				"addr":     config.MustEnv("CACHE_REDIS_ADDR", redisAddr()),
				"password": config.MustEnv("CACHE_REDIS_PASSWORD", config.MustEnv("REDIS_PASSWORD", "")),
				"db":       config.MustEnv("CACHE_REDIS_DB", 1),
			},
			{{- end}}
		},
	})
}
