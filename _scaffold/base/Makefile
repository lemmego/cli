dev:
	@lemmego dev

run:
	@lemmego run

build:
	@lemmego build

deps:
	@go mod tidy
	@go install github.com/a-h/templ/cmd/templ@latest
	@go install github.com/air-verse/air@latest

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
