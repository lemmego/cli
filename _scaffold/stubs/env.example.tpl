APP_NAME=Lemmego
APP_URL=http://localhost:8080
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080

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
{{- else}}
DB_HOST=localhost
DB_PORT={{.SQLDriver.DefaultPort}}
DB_DATABASE=lemmego
DB_USERNAME=
DB_PASSWORD=
{{- end}}
{{- end}}

# Logging. Every one of these has a working default, so the whole block can be
# deleted — and an empty value is treated as absent rather than as a choice.
#
# LOG_CHANNEL: console (readable lines on stderr), file (JSON in a rotating
# file), json (JSON on stderr, for a platform that collects stdio), stack
# (console and file together) or null. Defaults to console in development and
# stack in production.
LOG_CHANNEL=console
LOG_LEVEL=debug
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
