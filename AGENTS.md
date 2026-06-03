# heroldo

Discord bot that ingests HTTP multipart form data and relays it to Discord channels via `discordgo`.

## Architecture

```
HTTP multipart POST  →  buffered chan heroldo.Request  →  N worker goroutines → discordgo Session
```

- **`cmd/heroldo/`** — main package; HTTP handler (`http.go`), Discord sender workers (`discord.go`)
- **`pkg/heroldo/`** — shared types (`Request`, `File`)

`cmd/heroldo/heroldo.go` is a skeleton — `func main()` is empty (the package does not compile).

## Current state

- **git:** `master` branch, 2 commits (`7961472` initial, `b46eb15` content semantics). Working tree clean. Local `master` 1 ahead of `origin/master`.
- **Build:** `go build ./...` fails — empty `main()`.
- **Deps:** `go.mod` lists `discordgo`, `gonanoid`, `snowflake` etc., all marked `// indirect` (will fix after `main()` compiles). Run `go mod tidy` after implementing `main()`.
- **Tests:** None.

## TODO (inferred)

- Implement `main()`: init `discordgo.Session`, create `DiscordSender`, register `RequestHandler`, HTTP server lifecycle, signal handling.
- Add `.gitignore` to exclude `heroldo.exe`.
- Run `go mod tidy` after compilation succeeds.
- Add tests, CI, README, config.

## Commands

| What | Command |
|---|---|
| Build | `go build ./...` |
| Tidy | `go mod tidy` |
| Vet | `go vet ./...` |
| Format | `gofmt -s -w .` |

If tests are added later, run them with `go test ./...`.
