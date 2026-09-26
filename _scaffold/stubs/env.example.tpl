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

FILESYSTEM_DISK={{.FilesystemDisk}}
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
{{- end}}
{{- if .UsesRedis}}

# One connection, shared by everything that uses Redis.
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
{{- end}}
