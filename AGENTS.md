# Repository Guidelines

## Project Structure & Module Organization
- `cmd/tunman/main.go` is the single executable entrypoint (TUI by default, CLI subcommands when args are passed).
- `internal/ui/` contains Bubble Tea page models and rendering logic (`list`, `create`, `logs`, `help`, app shell).
- `internal/core/` contains runtime services (tunnel lifecycle control and JSON-backed storage).
- `internal/model/` defines shared domain types (tunnel, config, flexible time parsing).
- `docs/` holds product and setup docs; runtime data is stored outside the repo in `~/.tunman/`.

## Build, Test, and Development Commands
- `go build -o tunman ./cmd/tunman` builds the local binary.
- `go run ./cmd/tunman` starts the interactive TUI.
- `go run ./cmd/tunman list` runs a CLI command path.
- `go test ./...` runs all Go tests (currently no `_test.go` files are committed).
- `go fmt ./... && go vet ./...` formats and checks code before opening a PR.

## Coding Style & Naming Conventions
- Follow `.editorconfig`: Go files use tabs, Markdown/JSON/YAML use 2 spaces, UTF-8, LF, final newline.
- Use idiomatic Go naming: exported identifiers in `CamelCase`, package-private in `camelCase`, package names lowercase.
- Keep files and packages cohesive by layer (`ui`, `core`, `model`); avoid cross-layer shortcuts.
- Prefer small, explicit functions and `error` returns over panics.

## Testing Guidelines
- Place tests next to implementation using `*_test.go` (for example, `internal/core/storage_test.go`).
- Prefer table-driven tests for parsing, state transitions, and command argument handling.
- Cover core behaviors first: storage read/write, status sync, and controller start/stop edge cases.
- Run `go test ./...` locally before every push.

## Commit & Pull Request Guidelines
- Current branch history is empty, so no existing commit convention can be inferred.
- Use clear, imperative commit subjects; Conventional Commits are recommended (for example, `feat: add tunnel status sync`).
- Keep commits scoped to one concern and include rationale in the body when behavior changes.
- PRs should include: summary, testing notes (`go test ./...`, manual TUI/CLI checks), linked issue, and screenshots/GIFs for UI changes.

## Security & Configuration Tips
- Do not commit secrets, Cloudflare credentials, or generated files from `~/.tunman/`.
- Verify external dependencies (`cloudflared`, `screen`) are installed before runtime testing.
- Keep `config.json` and `tunnels.json` examples sanitized in docs and PR descriptions.

## Assistant Disclosure Rule
- For every response, if any skill or rule is triggered in the current turn, the first line must disclose all triggered items.
- Output format for the first line: `Triggered: skills=[<comma-separated list or none>]; rules=[<comma-separated list or none>]`.
- `skills` includes project and global Codex skills used in this turn.
- `rules` includes this `AGENTS.md` and any additional explicitly triggered rule files/instructions used in this turn.
