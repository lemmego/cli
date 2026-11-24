dev:
	@lemmego dev

run:
	@lemmego run

build:
	@lemmego build

test:
	@# APPTEST_POSTGRES_DSN is what makes the tests that need a database run
	@# rather than skip — and a skipped suite and a passing suite have the
	@# same exit code, so a run that finishes suspiciously fast is the
	@# symptom of this being unset.
	@if [ -z "$$APPTEST_POSTGRES_DSN" ]; then \
		echo "note: APPTEST_POSTGRES_DSN is unset, so tests needing a database will skip."; \
		echo "      export APPTEST_POSTGRES_DSN=\"postgres://$$USER@127.0.0.1:5432/postgres?sslmode=disable\""; \
	fi
	@go test ./...

deps:
	@go mod tidy
	@go install github.com/a-h/templ/cmd/templ@latest
	@go install github.com/air-verse/air@latest

appkey:
	@lemmego run appkey

migrate:
	@lemmego run migrate up

rollback:
	@lemmego run migrate down

migration:
	@lemmego run migrate create $(n)

handlers:
	@lemmego g handlers $(n)

input:
	@lemmego g input $(n)

model:
	@lemmego g model $(n)

form:
	@lemmego g form $(n)

ssr-start:
	@lemmego inertia-ssr start

ssr-stop:
	@lemmego inertia-ssr stop

ssr-check:
	@lemmego inertia-ssr check

libsql:
	@# The data directory belongs to the server, not the application: a project
	@# configured for libSQL knows only DB_URL. So this target is the one place
	@# that decides where a local database lives, and `make dev` starts the same
	@# server automatically -- this is for running it on its own.
	@#
	@# An absolute path rather than ./storage: --db-path is relative to sqld's
	@# working directory, so a relative path silently creates a second, empty
	@# database whenever sqld is started from anywhere else, and every table
	@# then looks as though it was never migrated.
	@#
	@# tr -d '\042\047' strips any quotes around a .env value using octal
	@# escapes, because a bracket expression holding a quote character needs
	@# nested shell quoting that make and sh disagree about.
	@if ! command -v sqld >/dev/null 2>&1; then \
		echo "error: sqld is not installed."; \
		echo "       brew install libsql    # or see github.com/tursodatabase/libsql"; \
		exit 1; \
	fi
	@driver=$$(grep -E '^DB_DRIVER=' .env 2>/dev/null | cut -d= -f2- | tr -d '\042\047'); \
	if [ -n "$$driver" ] && [ "$$driver" != libsql ]; then \
		echo "note: DB_DRIVER is $$driver, so the application will not use this server."; \
	fi; \
	addr=$$(grep -E '^DB_URL=' .env 2>/dev/null | cut -d= -f2- | tr -d '\042\047' \
		| sed -e 's|^https*://||' -e 's|/.*$$||'); \
	addr=$${addr:-127.0.0.1:8080}; \
	echo "libSQL on http://$$addr, data in $(CURDIR)/storage/data/libsql"; \
	exec sqld --db-path "$(CURDIR)/storage/data/libsql" --http-listen-addr "$$addr"
