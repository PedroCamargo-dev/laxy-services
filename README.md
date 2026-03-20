# laxy-services

A lightweight, keyboard-driven TUI for managing systemd services on Linux — built for speed and minimal resource usage on production servers.

```
╔══════════════════════════════════════════════════════════════════════════════════╗
║                            LAXY-SERVICES  (TUI)                                ║
╚══════════════════════════════════════════════════════════════════════════════════╝
┌─ SERVICES LIST (226)──────────────────────┐┌─ LIVE LOGS ● nginx  [LIVE]───────────────┐
│  NAME                  STATUS     SUB     ││2026-03-20T01:51:00 systemd[1]: Starting  │
│  ──────────────────────────────────────── ││2026-03-20T01:51:01 nginx: worker process │
│✓ nginx                 active     running ││2026-03-20T01:51:02 INFO  GET /api/v1/data│
│▶ chronyd               active     running ││2026-03-20T01:51:03 WARN  conn closed     │
│✗ docker                failed     dead    ││                                          │
│○ bluetooth             inactive   dead    ││                                          │
└───────────────────────────────────────────┘└──────────────────────────────────────────┘
 [↑↓/jk] navigate  [Enter/Tab] focus logs  [S] start  [X] stop  [R] restart  [Q] quit
```

## Features

- **Direct DBus communication** — talks to systemd via `go-systemd/dbus`, never shells out to `systemctl` for status or management
- **Real-time log streaming** — follows `journald` for the selected service with automatic lifecycle management (switching services kills the previous stream instantly)
- **In-log search** — `/` to search, `n`/`N` to navigate matches, inline highlight of results
- **Live/paused scroll** — full scroll history with `↑↓`, `PgUp/PgDn`, `g`/`G`; toggle live follow with `f`; indicator shows `[LIVE]` or `[PAUSED ↑+N]`
- **Instant service search** — just start typing to filter the list by name or description
- **Mouse support** — click to select services, scroll wheel works on both panels
- **Sorted list** — services ordered by state priority (`active` → `activating` → `failed` → `inactive`), then alphabetically
- **Zero CPU when idle** — log lines arrive via blocking channel reads, no polling

## Requirements

- Linux with systemd
- `journalctl` available in `$PATH`
- Go 1.21+ (to build from source)
- D-Bus system socket access (typically requires `sudo` or membership in the `systemd-journal` group for logs)

## Installation

### One-liner (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/PedroCamargo-dev/laxy-services/main/install.sh | sh
```

Downloads and installs the latest pre-built binary for your architecture (`amd64` or `arm64`) to `/usr/local/bin/laxy-services`. Override the install directory with `INSTALL_DIR=/your/path`.

### Download manually

Grab a pre-built binary from the [Releases](https://github.com/PedroCamargo-dev/laxy-services/releases/latest) page:

| Platform | File |
|----------|------|
| Linux x86-64 | `laxy-services-linux-amd64` |
| Linux ARM64  | `laxy-services-linux-arm64` |
| macOS Intel  | `laxy-services-darwin-amd64` |
| macOS Apple Silicon | `laxy-services-darwin-arm64` |

> **Note:** macOS binaries are provided for completeness but will not function — systemd and D-Bus are Linux-only. The tool requires a Linux system with systemd to operate.

```bash
chmod +x laxy-services-linux-amd64
sudo mv laxy-services-linux-amd64 /usr/local/bin/laxy-services
```

### From source

```bash
git clone https://github.com/PedroCamargo-dev/laxy-services.git
cd laxy-services
go build -o laxy-services ./cmd/laxy-services/
sudo mv laxy-services /usr/local/bin/
```

### Run directly

```bash
go run ./cmd/laxy-services/
```

> **Note:** managing services (start/stop/restart) requires root privileges or appropriate polkit rules. Run with `sudo` if needed.

## Key Bindings

### Service List (default focus)

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `Tab` | Focus the log panel |
| `S` | Start selected service |
| `X` | Stop selected service |
| `R` | Restart selected service |
| `Shift+R` | Refresh service list |
| *any key* | Add to search filter |
| `Backspace` | Remove last search character |
| `ESC` | Clear search filter |
| `Q` | Quit |

### Log Panel (after `Enter` / `Tab`)

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up one line |
| `↓` / `j` | Scroll down one line |
| `PgUp` / `PgDn` | Scroll up / down one page |
| `g` | Jump to oldest log line |
| `G` | Jump to latest (re-enable live) |
| `f` | Toggle live follow on/off |
| `/` | Enter log search mode |
| `S` / `X` / `R` | Start / Stop / Restart (works here too) |
| `Tab` / `ESC` | Return focus to service list |
| `Q` | Quit |

### Log Search Mode (after `/`)

| Key | Action |
|-----|--------|
| *type* | Extend search query |
| `n` / `Enter` | Jump to next match |
| `N` | Jump to previous match |
| `Backspace` | Remove last character |
| `ESC` | Clear search and exit search mode |

### Mouse

| Action | Effect |
|--------|--------|
| Left click on service | Select and start following its logs |
| Left click on log panel | Focus the log panel |
| Scroll wheel (list) | Move cursor up / down |
| Scroll wheel (logs) | Scroll log history |

## Architecture

```
laxy-services/
├── cmd/laxy-services/     # Entry point — initialises the TUI
└── internal/
    ├── models/            # Shared domain type: Service
    ├── systemd/           # DBus client: list units, start/stop/restart
    ├── journal/           # Async journald follower with context cancellation
    └── tui/
        ├── model.go       # Bubble Tea Model, Update loop, key & mouse handlers
        ├── view.go        # Panel rendering, log highlighting, status bar
        └── styles.go      # Lipgloss colour palette and style definitions
```

**Design principles:**
- No `os.Exec` for systemd interaction — DBus only
- No polling — log lines flow through a buffered channel; CPU idles at zero between events
- Goroutine lifecycle is explicit: each follower owns a `context.CancelFunc`, cancelled on service switch or quit
- No comments — naming is the documentation

## Stack

| | |
|-|-|
| Language | Go 1.24 |
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) v0.27 |
| Styling | [Lipgloss](https://github.com/charmbracelet/lipgloss) v1.0 |
| systemd / DBus | [go-systemd](https://github.com/coreos/go-systemd) v22 |

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for significant changes.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes
4. Open a pull request

## License

[MIT](LICENSE) — © 2026 Pedro Camargo
