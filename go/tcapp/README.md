# tcapp

Minimal Go HTTP service boilerplate.

- `main.go` only loads config, sets up the logger and signal handling, then calls `app.New(...).Run(ctx)`.
- `app/` contains the actual app, so integration/e2e tests can start it in-process
  (use `PORT=0`, wait on `app.Ready()`, read `app.Addr()` — see `app/app_test.go`).
- Logging: stdlib `log/slog` (JSON to stdout).
- Config: environment variables via [caarlos0/env](https://github.com/caarlos0/env).

## Configuration

| Variable           | Required | Default | Description                              |
|--------------------|----------|---------|------------------------------------------|
| `PORT`             | yes      | —       | HTTP listen port (`0` = random free port) |
| `LOG_LEVEL`        | no       | `info`  | `debug`, `info`, `warn`, `error`          |

For local runs, put overrides in `tcapp.env` (git-ignored); the Taskfile loads it automatically.

## Endpoints

```
$ curl -i localhost:8080/health
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

## Development

Requires Go 1.26+, [Task](https://taskfile.dev) and [golangci-lint v2](https://golangci-lint.run).

```sh
task run          # go run . (PORT defaults to 8080)
task test         # go test -race ./...
task lint         # golangci-lint run
task fmt          # gofumpt + goimports
task build        # ./bin/tcapp
task docker:local # image for the host platform, loaded into docker
task docker:push IMAGE=ghcr.io/you/tcapp:dev  # multi-arch (amd64 + arm64) build, pushed to a registry
```

## CI

TBD. tcapp currently lives inside the `playground` monorepo, so it has no GitHub Actions workflows of its own.
Planned once it's split out (or wired at the repo root):

- **PR**: build, test (`-race`), golangci-lint.
- **Tag `v*`**: build and push a multi-arch (amd64 + arm64) image to ghcr.io using `docker/build-push-action`.
  The Dockerfile already cross-compiles natively (`--platform=$BUILDPLATFORM`), so no QEMU is needed.
