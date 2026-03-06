# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TunMan is a Go TUI/CLI tool for managing Cloudflare Tunnels. It runs tunnels inside GNU Screen sessions and persists tunnel definitions as JSON in `~/.tunman/`. The UI is built with Bubble Tea (charmbracelet). The project is a v2.0 Go rewrite, currently work-in-progress.

## Build & Development Commands

```bash
make build              # Build binary to bin/tunman (with version/build-time ldflags)
make run ARGS="list"    # Build and run with arguments
make test               # go test -v ./...
make fmt                # go fmt ./...
make vet                # go vet ./...
make deps               # go mod download && go mod tidy
make install-dev        # Build then copy to ~/.local/bin/
go test -v -run TestName ./internal/core/  # Run a single test
```

Run `go fmt ./... && go vet ./...` before committing.

## Architecture

### Layers

- **`cmd/tunman/main.go`** — Single entrypoint. No args = TUI mode (Bubble Tea); with args = CLI subcommands (`list`, `start`, `stop`, `status`, `version`).
- **`internal/ui/`** — Bubble Tea UI layer. `app.go` is the root Model that owns page state and delegates to sub-models (`ListModel`, `CreateModel`, `DeleteModel`, `LogsModel`, `HelpModel`). Pages are selected via `Page` enum. Each sub-model follows the Bubble Tea `Update`/`View` pattern but receives shared state (tunnels, storage, controller, dimensions) as parameters rather than owning it.
- **`internal/core/`** — Runtime services:
  - `controller.go` — Tunnel lifecycle (setup/teardown via `cloudflared` CLI, start/stop via `screen` sessions, status sync, log retrieval, editor integration).
  - `storage.go` — JSON-file persistence in `~/.tunman/` (`config.json`, `tunnels.json`, per-tunnel logs and config dirs).
- **`internal/model/`** — Domain types: `Tunnel` (with Category: custom/testing/ephemeral and runtime Status), `Config`, `FlexTime` (multi-format time parser).
- **`internal/i18n/`** — Embedded YAML-based i18n (zh/en). Uses `//go:embed locales/*.yaml`. Language detected from `TUNMAN_LANG`, `LANG`, or `LC_ALL` env vars; defaults to Chinese (`zh`).

### Key Design Patterns

- Tunnels run inside GNU Screen sessions named `tunman-<id>`. The Controller syncs runtime status by checking screen session existence — `Status`, `PID`, and `SessionName` are not persisted (tagged `json:"-"`).
- Three tunnel categories with different lifecycles: **custom** and **testing** require cloudflared tunnel create/DNS setup; **ephemeral** skips setup and uses `cloudflared tunnel --url` directly.
- The app shell (`app.go`) manages a periodic tick for auto-refresh and a notification system with timestamped clear messages.

## External Dependencies

Runtime: `cloudflared` (Cloudflare Tunnel CLI) and `screen` (GNU Screen) must be installed on the host. The app checks for these at startup and shows warnings in the TUI.

## Coding Conventions

- Follow `.editorconfig`: Go files use tabs, Markdown/JSON/YAML use 2 spaces, UTF-8, LF endings.
- Code comments and UI strings are in Chinese; use the i18n system (`i18n.T`/`i18n.Tf`) for all user-facing strings. Add keys to both `internal/i18n/locales/zh.yaml` and `en.yaml`.
- Place tests next to implementation as `*_test.go`. Prefer table-driven tests.
- Use Conventional Commits (e.g., `feat:`, `fix:`, `refactor:`).
