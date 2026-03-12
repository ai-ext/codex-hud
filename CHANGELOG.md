# Changelog

## [0.2.0] - 2026-03-13

### New Features
- **Live usage (rate limits)** — real-time 5h/7d usage via WHAM API, matching Codex `/status`
- **Auto session switching** — HUD automatically detects and switches to new Codex sessions
- **tmux auto-launch** — creates a new tmux session with split panes when tmux is available but not in a session
- **Windows Terminal support** — `wt split-pane` for native Windows split experience
- **Pre-loading** — existing session data loaded instantly before TUI starts (no jumpy startup)
- **Windows install script** — `install.bat` / `install.ps1` for one-click PATH setup
- **`make install` / `make uninstall`** — quick install to `/usr/local/bin`

### Bug Fixes
- **Stale context on new session** — old session showing 45% context when starting a new session. Fixed by resetting per-session state when session ID changes
- **Usage stale flash** — old rate limit data (1%/4%) flashing briefly on startup before live API data arrives. Fixed by removing session file rate limit fallback; Usage card now only appears after WHAM API responds
- **TailFile duplication** — pre-loaded lines being re-processed through the channel. Fixed with `TailFileFromEnd()` that seeks to end of file
- **Wrapper mode not working** — `codex-hud` just launched codex without the HUD on macOS. Fixed by auto-detecting tmux and creating split sessions
- **Default mode confusion** — restored wrapper mode (codex + HUD together) as default; `--watch` for HUD-only
- **Rate limits null handling** — newer Codex versions (v0.114.0+) send `rate_limits: null` in token_count events, which was clearing existing rate limit data
- **Context token display** — now shows raw token values (e.g., `8,590 / 258,400`) matching Codex `/status` output

### Changes
- Rate limits are now fetched exclusively from the WHAM API (no longer from session file data)
- `--fresh` flag added (used internally by wrapper mode to skip old session pre-loading)
- Simplified fallback launcher — prints guidance message instead of attempting OS-specific terminal launches

## [0.1.0] - 2026-03-12

### Initial Release
- Real-time session monitoring from Codex JSONL files
- Context window usage with progress bar
- Token stats (input, cached, output, reasoning)
- Session info (duration, turns, working directory)
- Activity tracking (active/completed tool calls)
- Git status display
- Configurable via `~/.codex/codex-hud.toml`
- Cross-platform builds (macOS, Linux, Windows)
- tmux split-pane integration
- Auto-detection of latest session file
