APP_NAME=Lemmego
APP_URL=http://localhost:8080
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080

# Filled in by `make appkey`, which signs sessions and cookies. Empty is not a
# working value — MustEnv treats a present-but-empty variable as a decision, so
# an application left like this cannot authenticate anybody. The line exists
# empty rather than absent so that appkey has something to replace: it used to
# match any line mentioning APP_KEY and would overwrite the comment explaining
# it.
APP_KEY=

# Left commented out on purpose. MustEnv falls back only when a variable is
# absent, so setting this to an empty value would defeat the APP_KEY fallback
# rather than leaving it in place.
#JWT_SECRET=
{{- if .HasDatabase}}

# The application reads DB_CONNECTION; `lemmego run migrate` reads DB_DRIVER.
# Both are needed.
DB_CONNECTION={{.SQLDriver.ConnectionName}}
DB_DRIVER={{.SQLDriver.DriverName}}
{{- if eq .SQLDriver "sqlite"}}
DB_DATABASE=./storage/database.sqlite
{{- else if eq .SQLDriver "libsql"}}
# A libSQL server is addressed by URL, so there is no DB_HOST or DB_PORT.
DB_URL=http://127.0.0.1:8080

# And DATABASE_URL, because the two paths disagree about what a DSN is.
# `lemmego run migrate` builds its own from DB_HOST and friends, and
# migration.DataSource cannot express a URL at all -- it emits a sqlite file
# path, a MySQL user:pass@tcp(...) or Postgres keywords, and deliberately
# refuses libsql so it cannot quietly open a local FILE instead of the server.
# The migrator reads DATABASE_URL before it tries to build anything, so that
# is how it finds a libSQL server. Keep this in step with DB_URL above.
DATABASE_URL=http://127.0.0.1:8080
{{- else}}
DB_HOST=localhost
DB_PORT={{.SQLDriver.DefaultPort}}
DB_DATABASE=lemmego
DB_USERNAME=
DB_PASSWORD=
{{- if eq .SQLDriver "postgres"}}
# lib/pq defaults to sslmode=require and a local Postgres rarely serves TLS.
# Raise this to require or verify-full wherever the database is not local.
DB_SSLMODE=disable
{{- end}}

# `lemmego run migrate` builds its own DSN and reads this, where the
# application reads the options map in internal/configs/database.go. They are
# separate paths, so a parameter needed by both goes in both places.
DB_PARAMS={{if eq .SQLDriver "postgres"}}sslmode=disable{{end}}
{{- end}}
{{- end}}

# Logging. Every one of these has a working default, so the whole block can be
# deleted — and an empty value is treated as absent rather than as a choice.
#
# LOG_CHANNEL: console (readable lines, split across stdout and stderr), file
# (JSON in a rotating file), json (JSON on stderr, for a platform that collects
# stdio), stack (console and file together) or null. Defaults to console in
# development and stack in production.
LOG_CHANNEL=console
LOG_LEVEL=debug
# LOG_SPLIT_STREAMS: warn and above to stderr, everything below to stdout.
#
# On by default for a server and off for a CLI command, which is almost always
# right: a container platform classifies a line by the stream it arrived on, so
# a server writing info to stderr has every routine line recorded as an error,
# while a command's stdout is the output somebody ran it for.
#
# Set it to true for a worker or a scheduler — a long-running server that
# happens to be spelled as a subcommand, where the default answers the wrong
# question. The same case LOG_QUIET exists for.
# LOG_SPLIT_STREAMS=true
LOG_FILE=storage/logs/app.log
# Rotate on size as well as date, so a loop cannot fill the disk overnight.
LOG_DAILY=true
LOG_MAX_SIZE=33554432
LOG_MAX_FILES=7

FILESYSTEM_DISK={{.FilesystemDisk}}

# Where uploads live. Absolute in a real deployment: a relative path resolves
# against the working directory, so a single binary started from anywhere else
# writes its uploads somewhere else and serves nothing, with a 404 as the only
# symptom. The fs provider logs the root it resolved at boot.
#
# Two disks are rooted under it: app/public is served in full at /storage/,
# app/private is not served and is the default.
STORAGE_PATH=./storage
{{- if eq .FilesystemDisk "s3"}}
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_DEFAULT_REGION=us-east-1
AWS_BUCKET=
{{- end}}

SESSION_DRIVER={{.SessionDriver}}
{{- if .HasCache}}

CACHE_DRIVER={{.CacheDriver}}
CACHE_TTL=3600
{{- end}}
{{- if .HasQueue}}

TASKER_DRIVER={{.QueueDriver}}
{{- if .EnableAuth}}
# Who may open the queue dashboard: a comma-separated list of email
# addresses. Empty means nobody, which is the safe default — the dashboard can
# retry, cancel and delete jobs.
TASKER_ADMINS=
{{- end}}
{{- end}}
{{- if .UsesRedis}}

# One connection, shared by everything that uses Redis.
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
{{- end}}
{{- if .InertiaProvider}}

# Server-side rendering, overriding what bootstrap/providers.go configured.
#
# Leave it unset and the application decides. Set it to false to work on a
# page without a renderer in the way; set it to true to check that a page
# really does render on the server before shipping it.
#
# In development the renderer is the Vite dev server, so `lemmego dev` starts
# nothing extra. In production `lemmego inertia-ssr start` runs the built
# bundle.
INERTIA_SSR=
{{- end}}
{{- if .HasModule "mail"}}

# Mail. With no MAIL_HOST the log transport stays in place, which writes the
# envelope and — outside production — the rendered body, so a verification
# link is readable during development without a relay.
MAIL_HOST=
MAIL_PORT=587
MAIL_USERNAME=
MAIL_PASSWORD=
MAIL_FROM_ADDRESS=hello@example.com
MAIL_FROM_NAME={{.Name}}
{{- end}}
{{- if .HasModule "media"}}

# The disk renditions are written to. public is served at /storage/, which is
# what an <img> needs; put originals somewhere unserved if you keep them.
MEDIA_DISK=public
{{- end}}
{{- if .HasModule "search"}}

# postgres searches the tsvector columns this application already has and
# needs nothing else running. meilisearch keeps its own index and must be
# reachable at boot. See internal/configs/search.go.
SEARCH_ENGINE=postgres
SEARCH_PREFIX={{.Name}}_
SEARCH_MEILISEARCH_URL=http://127.0.0.1:7700
SEARCH_MEILISEARCH_KEY=
{{- end}}
{{- if .HasModule "pay"}}

# Stripe. Livemode is derived from the secret key's own prefix, so there is no
# flag here that could disagree with the key in use.
#
# Two webhook secrets, so one can be rotated without an interval in which
# events are refused — and a refused event is money moving with nothing
# recording it. `stripe listen --forward-to localhost:8080/pay/webhook`
# prints one for development.
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_WEBHOOK_SECRET_PREVIOUS=
{{- end}}
{{- if .HasModule "broadcast"}}

# memory for one process, redis for more than one. An unreachable Redis fails
# the boot rather than falling back, because a fallback passes staging — where
# there is one replica and every subscriber is local — and then delivers half
# the events in production.
BROADCAST_BUS=memory
{{- end}}
{{- if .HasModule "observe"}}

# Off unless this says otherwise, because an entry holds request headers,
# bound parameters and SQL.
OBSERVE_ENABLED=false
# The dashboard is at /observe and an empty token means nobody gets in,
# which is deliberate: it serves every statement this application has run.
OBSERVE_TOKEN=
# The fraction of requests recorded, 0 to 1. Leave it at 1 in development.
OBSERVE_SAMPLE=1
{{- end}}
{{- if .HasModule "i18n"}}

# Where the catalogue lives. The provider searches upward from the working
# directory, so `go test ./internal/...` finds it too.
I18N_PATH=resources/lang
{{- end}}
