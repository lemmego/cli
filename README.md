# lemmego/cli

[![Coverage Status](https://coveralls.io/repos/github/lemmego/cli/badge.svg?branch=main)](https://coveralls.io/github/lemmego/cli?branch=main)

The `lemmego` binary. It generates a new project, generates code inside one,
and runs the development loop for it.

```sh
lemmego new my-app --non-interactive --module github.com/me/my-app
cd my-app
lemmego run migrate up
lemmego dev
```

Everything below is what the binary actually does today. Where a command's
behaviour is surprising, the reason is given rather than left to be
rediscovered.

## Installation

**macOS and Linux**

```sh
curl -fsSL https://raw.githubusercontent.com/lemmego/cli/refs/heads/main/installer.sh | sudo sh
```

**Windows**

```powershell
irm https://raw.githubusercontent.com/lemmego/cli/refs/heads/main/installer.ps1 | iex
```

Both resolve the newest GitHub release and take the same two overrides, so
neither script needs editing when a version ships:

| Variable | Default | Meaning |
|---|---|---|
| `LEMMEGO_VERSION` | the latest release | Pin a version, e.g. `v0.4.1` |
| `LEMMEGO_INSTALL_DIR` | `/usr/local/bin`, `C:\Windows\System32` | Where the binary lands |

**From source**

```sh
go install github.com/lemmego/cli/cmd@latest
```

This installs a binary called `cmd`, because that is the package's directory
name — rename it to `lemmego`. A binary built this way reports its module
version; a binary built from a checkout reports `dev+<commit>`. Neither is a
hardcoded string, deliberately: one used to live in `root.go` and went stale
at `0.1.59`, so seven consecutive releases reported a number seven releases
old and nothing failed, because a hardcoded version is always valid.

```sh
lemmego --version
```

## `lemmego new` — create a project

```sh
lemmego new my-app
```

**It creates `./my-app`, not a project in the current directory.** The
directory is created if it does not exist and must be empty if it does.

Interactively, the first question is the module path and the second is how
much else you want to be asked:

- **Quick start** — SQLite, the Lemmego ORM, file cache, SQL queue, file
  sessions, local disk. You still choose the preset, whether to scaffold
  authentication, and the frontend.
- **Customise** — a driver for each part.

`--exp` adds one more question in custom mode, for GPA. It gates the question
only; `--gpa` works without it.

### Non-interactive

```sh
lemmego new my-app --non-interactive \
  --module github.com/me/my-app \
  --preset mvc --frontend inertia_react \
  --database postgres --orm orm \
  --cache redis --queue redis --session redis \
  --disk s3
```

`--module` is the one required flag. Everything else falls back to the same
defaults the interactive form preselects — one `defaultProjectConfig`
function feeds both paths, because they used to disagree: interactively the
default was whichever option happened to be listed first, and
non-interactively it was a bool flag's zero value, so the same command
produced different projects depending on how it was invoked.

| Flag | Values | Default |
|---|---|---|
| `--module` | a Go module path | **required** |
| `--preset` | `mvc`, `rest_api` | `mvc` |
| `--frontend` | `go_templates`, `templ`, `inertia_react`, `inertia_vue`, `templ_inertia_react`, `templ_inertia_vue` | `go_templates` |
| `--database` | `sqlite`, `mysql`, `postgres`, `none` | `sqlite` |
| `--orm` | `orm` (Lemmego), `gorm`, `bun`, `none` | `orm` |
| `--cache` | `file`, `memory`, `redis`, `none` | `file` |
| `--queue` | `sql`, `redis`, `none` | `sql` |
| `--session` | `file`, `memory`, `redis` | `file` |
| `--disk` | `local`, `s3` | `local` |
| `--auth` | bool | `true` |
| `--gpa` | bool | `false` |
| `--redis` | bool | `false` |
| `--exp` | bool | `false` |
| `--non-interactive` | bool | `false` |

`--redis` is shorthand for "use Redis for whatever I did not name". It moves
the *defaults* for cache, queue and sessions, so an explicit `--cache file`
alongside it still wins.

An unrecognised value is an error naming the accepted ones, not a silent
fallback.

### What one answer implies about another

- **A REST API ignores `--frontend`.** No frontend overlay is applied at
  all; the field is pinned to `go_templates` only so the stubs have a value
  to compare against.
- **No database means no ORM and no GPA.** The connector is what registers
  the connection, and GPA is registered by the connector.
- **There is no `--session none`.** The server builds its handler as
  `session.LoadAndSave(router)`, and the session carries the CSRF token,
  validation errors and flash messages — an application without one cannot
  render its own error pages.
- **`--cache none` is not the null store.** It leaves the cache out of the
  project entirely: no dependency, no provider, no config file. The cache
  module's `null` store is registered and configured and simply misses every
  read.
- **Redis is derived, never declared.** The connection block is written when
  some driver needs it. A standalone toggle used to do three unrelated jobs
  at once, which is why choosing the Redis session driver in a project
  scaffolded without the toggle used to panic on the first request: the
  driver was selected and the connection it reads was never written.

Three combinations are refused rather than scaffolded, because each one
builds cleanly and then fails at runtime: a SQL queue with no database,
authentication with no database, and GPA with no database.

### What `new` does after scaffolding

In order: renames the module throughout, formats the tree, copies
`.env.example` to `.env`, creates `storage/database.sqlite`, installs Node
modules and builds the frontend if the preset has one, runs `templ generate`
if the preset has templates, runs `go mod tidy`, and writes an `APP_KEY`.

The key is written here, with `crypto/rand` over the standard library, rather
than by shelling out to the project's own `appkey` command — that is how it
used to work and it could not work. Reaching that command goes through
`lemmego run`, which boots the generated application, and the application
cannot boot until somebody has filled in database credentials. So the step
printed "Generating app key" and then panicked with "DB Username is
required", and **every new project started life with no key at all**, which
means it could not authenticate anybody. The project keeps its own `appkey`
command for rotating the key later, when the database does work.

## `lemmego dev` — the development loop

```sh
lemmego dev
```

Starts Vite (if there is a `package.json`), then air, then `templ generate
--watch` (if there are `.templ` files), with each process's output prefixed
and coloured. Ctrl+C stops all of them.

**Vite goes first, and is waited for.** An Inertia project's server-side
renderer *is* the Vite dev server: `@inertiajs/vite` serves
`/__inertia_ssr`, and the address is published by writing `./public/hot`.
Starting air first meant the Go process booted with no hot file, so the first
page loads rendered against a standalone renderer that `dev` deliberately
never starts — and strict SSR, which is on outside production, turns that
into a 500 rather than a quiet degrade. A stale hot file from a previous run
is removed before the wait, because it would be believed instantly and point
at a dev server that is gone.

The wait is bounded at 20 seconds and is not fatal: a project may have a
`package.json` and no Vite, and the right answer then is to start the Go
server and let its own errors say so.

`lemmego dev` does **not** start the standalone Inertia SSR server. In
development the Vite dev server is the renderer; the standalone one is for
production and is skipped by the client bundle under
`import.meta.env.PROD`. See `inertia-ssr` below.

## `lemmego build` — build assets for a deploy

```sh
lemmego build
```

Runs `templ generate`, then the `build` npm script, then — only when the
project has an SSR entry point (`resources/js/ssr.{tsx,jsx,ts,js}`) **and** a
`build:ssr` script — the SSR build. Not running that second build is why "the
bundle has never been built" was the normal state of every Inertia project:
the documented build command never built it.

Last, and unconditionally, it deletes `public/hot`. That file points both the
asset tags and the server-side renderer at a Vite dev server; left behind in
a build it produces a deploy whose CSS and JS 404 and whose markup is never
server-rendered, with nothing in the logs to say why.

## `lemmego run` — run the application

```sh
lemmego run                 # start the HTTP server
lemmego run migrate up
lemmego run migrate down
lemmego run migrate create add_posts_table
lemmego run appkey
lemmego run tasker:schedule
```

`run` passes everything after it to `go run ./cmd/app`, so it reaches any
command the application or its providers register. Flags are not parsed by
the CLI, so the application's own flags arrive intact.

Frontend assets are rebuilt first **only for commands that serve HTTP**. A
bare `run`, or one whose first argument is not a subcommand, rebuilds; a
namespaced command (anything containing `:`) and `inspire`, `key`, `migrate`
and `rollback` do not.

## `lemmego gen` — generate code

Aliased to `g`. Every subcommand takes a name in `snake_case`, or `-i` to be
asked instead.

```sh
lemmego g handlers post      # internal/handlers/post_handlers.go
lemmego g model post         # internal/models/post.go
lemmego g input post         # internal/inputs/post_input.go
lemmego g migration create_posts_table
                             # internal/migrations/<version>_create_posts_table.go
lemmego g form post          # resources/js/Pages/Forms/Post.tsx
lemmego g form post -f templ # templates/post.templ
```

`-f` / `--flavor` is `react` or `templ`, and `react` is the default.

Interactively, `g -i model` asks for each field's name, type and attributes
(required, unique, primary) and understands relations; `id`, `created_at`,
`updated_at` and `deleted_at` are provided and refused as answers. `g -i
form` asks for the submit route and the fields.

### Where generated code goes

Not from constants in this repository. A project's paths live in
`internal/configs`, which is Go source with an `init` function — only the
project's own binary can evaluate it. So the CLI asks, by running
`go run ./cmd/app config:paths` and reading one line of JSON, which is what
makes an edited `app.go` actually move where a generated model lands.

That costs a compile, so the answer is cached for the rest of the run and
bounded at 90 seconds. Outside a project, or when the project does not build,
the conventional paths are used **and the CLI says so on stderr** — silence
would be worse, because a project that configured `./domain/models` and sees
a file in `./internal/models` has no way to tell whether the setting was
ignored or the question failed.

`MIGRATIONS_DIR` still overrides the migration path, because it is what the
`migrate` command itself reads and the two have to agree.

## `lemmego inertia-ssr` — the standalone renderer

For production, or for exercising server-side rendering without a Vite dev
server.

```sh
lemmego inertia-ssr start                    # runs bootstrap/ssr/ssr.js, writes .ssr.pid
lemmego inertia-ssr stop                     # SIGTERM, then SIGKILL if it is still there
lemmego inertia-ssr check [--host] [--port]  # asks it to render
```

**`check` renders rather than dials.** A Node process listening with a broken
module graph passes a liveness check and fails every render — and because
Inertia falls back to client-rendered HTML, the site answers 200 throughout.
So `check` POSTs to `/render` and reads the answer. The server's own `/health`
endpoint reports only that the process is up, which is the same thing a port
dial reports.

The component it asks for is deliberately one the bundle cannot have, so the
renderer is *expected* to refuse it — and the refusal is the signal: the
process is up, the module graph loaded, and the bundle's router ran far enough
to decide the component does not exist. **A 500 is therefore a pass.** Three
answers are not: an empty body (the bundle never loaded), a 404 (there is no
render endpoint at that address) and a connection error.

That makes it a shallow check. It cannot tell a renderer that fails only on
unknown components from one that fails on all of them, which is the condition
that matters most — proving a renderer produces documents needs a props-free
page to ask for, which is the application's to supply. `lemmego/inertia`'s
`WithSSRProbeComponent` is where that lives.

`start` takes no `--port` or `--host`, because `@inertiajs/core`'s server
reads neither: its port is fixed when the bundle is built, by the Vite
plugin's `inertia({ ssr: { port } })` option. `start` therefore prints a PID
and no address, since it does not choose one and cannot read the one the
bundle baked in. `check` is what answers which address is actually serving,
and defaults to `127.0.0.1:13714`.

## `lemmego cache-clean` — drop the scaffold cache

```sh
lemmego cache-clean
```

Removes `~/.cache/lemmego/scaffold`, so the next `lemmego new` uses the
scaffold embedded in the binary.

## The scaffold

`lemmego new` generates from `cli/_scaffold`, which is the only starter.
There is no example application in the monorepo; to see what a generated
project looks like, generate one.

```
_scaffold/
  VERSION      bumped in the same commit as any change below
  base/        every file every project gets
  overlays/    one per choice: six auth variants, six frontends, mvc, rest_api
  stubs/       templates rendered against the project's configuration
```

`stubs/versions.json` pins every `lemmego/*` module a generated project
depends on. A package missing from it floats to `latest`, which defeats the
pin for everybody.

**The CLI prefers the cached scaffold over its own embedded copy.** On each
`lemmego new` it checks `_scaffold/VERSION` on `main` against the cache's and
downloads if the remote is newer. The cache is then used whether or not it is
newer than the copy compiled into the binary, so a locally built CLI can
scaffold older templates than it carries — and local edits to `_scaffold`
are invisible. Set `LEMMEGO_SCAFFOLD_SOURCE=embedded` to test an edit, or run
`lemmego cache-clean`.

## Environment

| Variable | Read by | Effect |
|---|---|---|
| `LEMMEGO_SCAFFOLD_SOURCE` | `new` | `embedded` skips the download and the cache |
| `MIGRATIONS_DIR` | `g migration` | Overrides where migrations are written |
| `APP_NAME` | help output | The root command's short description |
| `LEMMEGO_SCAFFOLD_BUILD` | tests | Set to run the compile gate |
| `LEMMEGO_SCAFFOLD_REPLACE` | tests | A monorepo checkout to `replace` the pins with |

The binary autoloads `.env` from the working directory, which is how
`MIGRATIONS_DIR` and `APP_NAME` reach it from inside a project.

## Developing the CLI

```sh
go test ./...
```

The offline tests parse the generated tree: imports, unused identifiers, the
pins in `versions.json`, the preset matrix. They cannot catch everything —
`parser.ParseFile` accepts a duplicate import, because that is a type error
rather than a parse error, and that is exactly the bug that once shipped in
the templ+Inertia presets.

The gate that does catch it compiles the generated project, and is opt-in
because it is slow:

```sh
LEMMEGO_SCAFFOLD_BUILD=1 LEMMEGO_SCAFFOLD_REPLACE=$HOME/code/lemmego \
  go test -run TestGeneratedAuthProjectsCompile -timeout 30m ./...
```

`LEMMEGO_SCAFFOLD_REPLACE` points the generated `go.mod` at a monorepo
checkout, so the gate tests the working tree rather than the published tags.

End to end, which is the only check that answers whether `lemmego new`
works:

```sh
LEMMEGO_SCAFFOLD_SOURCE=embedded lemmego new throwaway \
  --non-interactive --module example.com/throwaway --frontend inertia_react
cd throwaway
go vet ./... && pnpm install && pnpm build && npx tsc --noEmit
lemmego run migrate up
lemmego dev
```

Then request the home page and assert the markup is server-rendered rather
than an empty `#app` — a failed render answers 200 with a client-rendered
container, so the HTTP status proves nothing.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to
discuss what you would like to change. Please update the tests, and bump
`_scaffold/VERSION` in the same commit as any `_scaffold` change — an
installed CLI will not pick the change up otherwise.

## License

[MIT](https://choosealicense.com/licenses/mit/)
