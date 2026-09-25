APP_NAME=Lemmego
APP_URL=http://localhost:8080
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080
# JWT_SECRET is commented out on purpose: MustEnv falls back only when the
# variable is absent, so an empty value here would defeat the APP_KEY
# fallback and leave auth unable to verify anything. Set it to override.
#JWT_SECRET=
DB_CONNECTION=sqlite
DB_DATABASE=./storage/database.sqlite
DB_DRIVER=sqlite
#DB_HOST=localhost
#DB_PORT=3306
#DB_USERNAME=root
#DB_PASSWORD=
FILESYSTEM_DISK=local
SESSION_DRIVER={{.SessionDriver}}
{{- if .EnableRedis}}
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
{{- end}}
