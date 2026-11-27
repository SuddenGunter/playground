# tcapp

Minimal Go HTTP service boilerplate backed by PostgreSQL.

- `main.go` only loads config, sets up the logger and signal handling, then calls `app.New(...).Run(ctx)`.
- `app/` contains the actual app. Integration tests start it in-process
  (use `PORT=0`, wait on `app.Ready()`, read `app.Addr()`; see `app/app_test.go`).
- Logging: stdlib `log/slog` (JSON to stdout).
- Config: environment variables via [caarlos0/env](https://github.com/caarlos0/env).
- Database: PostgreSQL via [pgx](https://github.com/jackc/pgx) (`pgxpool`). The schema (`mockdata/schema.sql`)
  is embedded in the binary and applied on startup with `CREATE TABLE IF NOT EXISTS`.

## Layout

```
main.go              entrypoint
app/                 wiring: DB pool, schema, all routes (routes()), HTTP server, graceful shutdown
app/config/          env config
api/health/          GET /health
api/respond/         JSON response helpers shared by handlers
api/mockdata/create/ POST /mock-data
api/mockdata/get/    GET /mock-data/{id}
api/mockdata/list/   GET /mock-data
mockdata/            mock_data table: model, schema, pgx store
internal/testenv/    testcontainers helpers for integration tests
```

## Configuration

| Variable           | Required | Default | Description                                     |
|--------------------|----------|---------|-------------------------------------------------|
| `PORT`             | yes      | n/a     | HTTP listen port (`0` = random free port)        |
| `DATABASE_URL`     | yes      | n/a     | Postgres connection string                      |
| `LOG_LEVEL`        | no       | `info`  | `debug`, `info`, `warn`, `error`                 |

For local runs, put overrides in `tcapp.env` (git-ignored); the Taskfile loads it automatically.

## Endpoints

| Method | Path               | Description                                           |
|--------|--------------------|-------------------------------------------------------|
| GET    | `/health`          | Liveness: `200 {"status":"ok"}` (doesn't touch the DB) |
| POST   | `/mock-data`       | Create a row. Body `{"name": "...", "value": "..."}` (`name` required, ≤ 256 chars; `value` ≤ 4096 chars) → `201` |
| GET    | `/mock-data`       | List rows ordered by id. `?limit=N` (1–500, default 50) |
| GET    | `/mock-data/{id}`  | Get one row → `200`, or `404` if it doesn't exist      |

Errors are JSON: `{"error": "..."}`.

Timestamps are always returned in UTC. Length limits are enforced by `CHECK` constraints in the schema.
The schema is applied with `CREATE TABLE IF NOT EXISTS`, so a local database created before a schema change
keeps the old table. Reset it with `task db:down -- -v && task db:up`.

```sh
curl -X POST localhost:8080/mock-data -d '{"name":"first","value":"hello"}'
# {"id":1,"name":"first","value":"hello","created_at":"2026-11-27T19:03:11.50052Z"}
curl localhost:8080/mock-data?limit=10
curl localhost:8080/mock-data/1
```

## Development

Requires Go 1.26+, [Task](https://taskfile.dev), [golangci-lint v2](https://golangci-lint.run),
and Docker or rootless podman.

```sh
task db:up             # local Postgres via compose (COMPOSE defaults to "podman compose")
task run               # go run . against the compose DB (PORT=8080)
task db:down           # stop it (task db:down -- -v also deletes its data)
task test              # unit tests only, no containers
task test:integration  # unit + integration tests, Postgres via testcontainers
task lint              # golangci-lint run (integration-tagged files included)
task fmt               # gofumpt + goimports
task build             # ./bin/tcapp
task docker:local      # image for the host platform, loaded into docker
task docker:push IMAGE=ghcr.io/you/tcapp:dev  # multi-arch (amd64 + arm64) build, pushed to a registry
```

### Tests

- **Unit tests** (`go test ./...`): config parsing, health handler, and the mock data handlers with an in-memory
  fake store.
- **Integration tests** (`//go:build integration`): each package starts one Postgres container in `TestMain`,
  and every test gets its own freshly created database (`testenv.Postgres.NewDatabase`), so tests can run in parallel.
  `app/app_test.go` runs the real app via `app.New(...).Run(ctx)`, seeds rows straight into the DB, then
  checks the HTTP API, shutdown and startup failures. `mockdata/store_test.go` tests the pgx store directly.

#### Podman notes

`task test:integration` defaults to rootless podman:

- `DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock`. The socket must be running:
  `systemctl --user enable --now podman.socket`.
- `TESTCONTAINERS_RYUK_CONTAINER_PRIVILEGED=true`. Rootless podman can't stop containers started through its API
  (`rootless netns: kill network process: permission denied`), so testcontainers' own terminate leaves stopped
  containers behind. Privileged Ryuk force-removes them when the tests finish.

With Docker, set `DOCKER_HOST` to your Docker socket (e.g. in `tcapp.env`) or run
`go test -tags integration ./...` directly.

## CI

TBD. tcapp currently lives inside the `playground` monorepo, so it has no GitHub Actions workflows of its own.
Planned once it's split out (or wired at the repo root):

- **PR**: build, unit + integration tests (`-race`; GitHub's Ubuntu runners have Docker), golangci-lint.
- **Tag `v*`**: build and push a multi-arch (amd64 + arm64) image to ghcr.io using `docker/build-push-action`.
  The Dockerfile already cross-compiles natively (`--platform=$BUILDPLATFORM`), so no QEMU is needed.
