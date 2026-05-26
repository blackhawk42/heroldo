# heroldo

Discord bot that ingests HTTP multipart form data and relays it to Discord channels via `discordgo`.

## Architecture

```
HTTP multipart POST  →  buffered chan heroldo.Request  →  N worker goroutines → discordgo Session
```

- **`cmd/heroldo/`** — main package; HTTP handler, Discord sender workers
- **`pkg/heroldo/`** — shared types (`Request`, `File`)

`cmd/heroldo/heroldo.go` is a skeleton — `func main()` is **not yet written** (the package does not compile).

## Current state

- No tests, no CI, no linter config, no Makefile, no README.
- No commits (`master` is empty — `proto/` is untracked).
- Go 1.26.1, module `github.com/r/blackhawk42/heroldo`.

## Commands

| What | Command |
|---|---|
| Build | `go build ./...` |
| Tidy | `go mod tidy` |
| Vet | `go vet ./...` |
| Format | `gofmt -s -w .` |

If tests are added later, run them with `go test ./...`.
