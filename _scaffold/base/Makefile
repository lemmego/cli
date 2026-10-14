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
