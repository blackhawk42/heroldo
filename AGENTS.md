# heroldo

Discord bot that ingests HTTP multipart form data and relays it to Discord channels via `discordgo`.

## Architecture

```
HTTP multipart POST  →  buffered chan heroldo.Request  →  N worker goroutines → discordgo Session
```

- **`cmd/heroldo/`** — main package; HTTP handler (`http.go`), Discord sender workers (`discord.go`), CLI entrypoint (`heroldo.go`)
- **`pkg/heroldo/`** — shared types (`Request`, `File`)
- **`pkg/set/`** — generic `Set[T comparable]` with `Intersection`, `Members` (Go 1.26 iter.Seq)

## Current state

- **git:** `master` branch, 13 commits. Working tree has uncommitted comment-only changes to `pkg/heroldo/heroldo.go`. Untracked `cmd/testscript/` (Python test harness). Local `master` 8 ahead of `origin/master`.
- **Build:** `go build ./...` succeeds. `go vet ./...` passes.
- **`main()`:** Fully implemented — cobra CLI with viper config (file, env `HEROLDO_*`, flags), signal handling (SIGINT/SIGTERM), graceful HTTP server + DiscordSender shutdown with configurable timeout.
- **Flags:** `--config/-f`, `--token/-t`, `--channels/-c`, `--port/-p` (default 8080), `--concurrency/-w` (default 5), `--max-body-size` (default 50 MB), `--shutdown-timeout` (default 30s).
- **`.gitignore`:** Present (Go template + Python + images). `heroldo.exe` excluded from git.
- **Deps:** `go.mod` lists `discordgo`, `gonanoid`, `cobra`, `viper`. `viper` transitive deps marked `// indirect`.
- **Tests:** None in Go. `cmd/testscript/` has an untracked Python test script (`test_heroldo.py`) with a test image.
- **CI, Makefile, Dockerfile, README:** None.

## Commands

| What | Command |
|---|---|
| Build | `go build ./...` |
| Tidy | `go mod tidy` |
| Vet | `go vet ./...` |
| Format | `gofmt -s -w .` |

If tests are added later, run them with `go test ./...`.
