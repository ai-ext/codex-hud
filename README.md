# codex-hud

Real-time monitoring HUD for [OpenAI Codex CLI](https://github.com/openai/codex). Runs alongside Codex in a split pane and shows live session stats.

```
╭──────────────────────────────────────────────────────────────────╮
│                                                                  │
│  ● gpt-5.4    high    untrusted    v0.114.0                     │
│  ╭──────────────────────────────────────────────────────────╮   │
│  │ Context                                                   │   │
│  │ ██████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ │   │
│  │ 44.5%  114,867 / 258,400 tokens                           │   │
│  ╰──────────────────────────────────────────────────────────╯   │
│  ╭────────────────────────╮╭────────────────────────╮           │
│  │ Tokens                  ││ Session                │           │
│  │ ↓ in 2.6M               ││ duration 20m 0s        │           │
│  │ ↻ cache 2.2M            ││ turns 3                │           │
│  │ ↑ out 14,142            ││ cwd ~                  │           │
│  │ ◆ reason 5,342          ││                        │           │
│  ╰────────────────────────╯╰────────────────────────╯           │
│  ╭──────────────────────────────────────────────────────────╮   │
│  │ Usage                                                     │   │
│  │ 5h  █████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 12%        │   │
│  │ 7d  ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 10%        │   │
│  ╰──────────────────────────────────────────────────────────╯   │
│  ╭──────────────────────────────────────────────────────────╮   │
│  │ Activity                                                  │   │
│  │ ▶ exec_command                                            │   │
│  │ exec_command x53                                          │   │
│  ╰──────────────────────────────────────────────────────────╯   │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

## Features

- **Context window** — real-time token usage with progress bar
- **Token stats** — input, cached, output, reasoning tokens
- **Usage (rate limits)** — live 5h/7d usage from WHAM API
- **Session info** — duration, turn count, working directory
- **Activity** — active/completed tool calls
- **Git status** — branch, dirty state (optional)
- **Auto session switching** — detects new Codex sessions automatically

## Install

### Pre-built binaries (Go 불필요)

[GitHub Releases](https://github.com/ai-ext/codex-hud/releases)에서 OS에 맞는 바이너리를 다운로드:

**macOS / Linux:**
```bash
# macOS (Apple Silicon)
curl -L -o codex-hud https://github.com/ai-ext/codex-hud/releases/latest/download/codex-hud-darwin-arm64
# macOS (Intel)
curl -L -o codex-hud https://github.com/ai-ext/codex-hud/releases/latest/download/codex-hud-darwin-amd64
# Linux
curl -L -o codex-hud https://github.com/ai-ext/codex-hud/releases/latest/download/codex-hud-linux-amd64

chmod +x codex-hud
sudo mv codex-hud /usr/local/bin/
```

**Windows:**
1. [codex-hud-windows-amd64.exe](https://github.com/ai-ext/codex-hud/releases/latest/download/codex-hud-windows-amd64.exe) 다운로드
2. `scripts/install.bat` 또는 `install.ps1`과 같은 폴더에 넣고 실행:
```powershell
# PowerShell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1

# 또는 CMD
scripts\install.bat
```
이 스크립트가 자동으로 `%LOCALAPPDATA%\codex-hud\`에 복사하고 PATH에 추가합니다.

### Build from source (Go 1.21+ 필요)

```bash
git clone https://github.com/ai-ext/codex-hud.git
cd codex-hud

# macOS
make build && make install

# Linux
go build -o dist/codex-hud ./cmd/codex-hud
sudo cp dist/codex-hud /usr/local/bin/

# Windows (PowerShell)
go build -o codex-hud.exe ./cmd/codex-hud
powershell -ExecutionPolicy Bypass -File scripts/install.ps1

# Cross-platform build (all 4 binaries)
make build-all
```

## Usage

### Default (wrapper mode)

```bash
codex-hud
```

Launches Codex + HUD together in a split pane. Automatically detects:
- **tmux** — splits the current pane (or creates a new session)
- **Windows Terminal** — uses `wt split-pane`
- **Other** — falls back to HUD-only watch mode

### Watch mode (HUD only)

```bash
codex-hud --watch
```

Runs only the HUD panel. Start Codex separately in another terminal. Useful when you already have Codex running.

### Options

```
--watch          HUD panel only (run codex separately)
--file <path>    Monitor a specific .jsonl session file
--split bottom   Split direction: bottom (default) or right
--version        Show version
```

## Configuration

Optional config file at `~/.codex/codex-hud.toml`:

```toml
[display]
theme = "default"
refresh_ms = 500
show_rate_limit = true
show_activity = true
show_git = true

[git]
show_dirty = true
show_ahead_behind = false
show_file_stats = false

[tmux]
auto_detect = true
position = "bottom"
size = 30
```

## How it works

1. Codex CLI writes session events to `~/.codex/sessions/*.jsonl`
2. codex-hud watches these files with fsnotify for real-time updates
3. Each `token_count` event contains context usage, token stats
4. Rate limits are fetched from the WHAM API (same as Codex `/status`)
5. The TUI renders everything using [Bubble Tea](https://github.com/charmbracelet/bubbletea)

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `r` | Refresh git status |

## Requirements

- [OpenAI Codex CLI](https://github.com/openai/codex) installed and authenticated
- Go 1.21+ (for building from source)

## Uninstall

```bash
# macOS/Linux
make uninstall

# Windows
powershell -ExecutionPolicy Bypass -File scripts/uninstall.ps1
```

## License

MIT
