# codex-hud Design Document

**Date:** 2026-03-07
**Status:** Approved

## Overview

Go로 만드는 크로스 플랫폼 Codex CLI HUD. `~/.codex/sessions/*.jsonl`을 실시간 감시하여 모델 정보, 컨텍스트 사용량, 토큰 통계, Git 상태, 도구 호출 현황을 터미널 TUI로 보여준다.

## Goals

- Windows (PowerShell, Git Bash), Linux, macOS 전부 단일 바이너리로 동작
- Codex 바이너리에 손대지 않음 (Passive File Watcher)
- 프론트 개발자가 봐도 예쁜 카드 기반 TUI
- `codex-hud` 한 번 실행으로 codex + HUD 동시 기동 (wrapper 모드)
- 각 세션마다 독립적으로 추적 (세션별 HUD)
- 환경 자동 감지: tmux / Windows Terminal / 일반 터미널

## Architecture

```
~/.codex/sessions/**/*.jsonl
         │
         │  fsnotify (file watch)
         ▼
    ┌──────────┐
    │  Parser   │  jsonl → Go structs
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │  State    │  in-memory session state
    └────┬─────┘
         │
         ▼
    ┌──────────┐
    │  TUI     │  bubbletea + lipgloss
    └──────────┘
```

### Data Flow

1. `codex-hud` 실행 → `~/.codex/sessions/`에서 오늘 날짜의 가장 최근 `.jsonl` 파일 자동 감지
2. fsnotify로 디렉토리 + 파일 감시
3. 새 줄 append 감지 → 파싱 → State 업데이트 → TUI 리렌더
4. 새 세션 파일 생성 감지 → 자동 전환

### Session Log Format (Codex v0.111.0)

#### Event Types

| type | description |
|------|-------------|
| `session_meta` | 세션 시작. id, cli_version, cwd, model_provider |
| `turn_context` | 턴 시작. model, reasoning_effort, approval_policy, sandbox_policy |
| `event_msg` (token_count) | 토큰 사용량. total/last token usage, model_context_window, rate_limits |
| `event_msg` (task_started) | 턴 시작 이벤트 |
| `event_msg` (task_complete) | 턴 완료 이벤트 |
| `event_msg` (agent_message) | 에이전트 메시지 |
| `response_item` (function_call) | 도구 호출 (name, arguments, call_id) |
| `response_item` (function_call_output) | 도구 결과 |

#### Token Count Structure

```json
{
  "type": "event_msg",
  "payload": {
    "type": "token_count",
    "info": {
      "total_token_usage": {
        "input_tokens": 1385034,
        "cached_input_tokens": 1270784,
        "output_tokens": 11636,
        "reasoning_output_tokens": 2548,
        "total_tokens": 1396670
      },
      "last_token_usage": {
        "input_tokens": 70285,
        "cached_input_tokens": 64384,
        "output_tokens": 615,
        "reasoning_output_tokens": 99,
        "total_tokens": 70900
      },
      "model_context_window": 258400
    },
    "rate_limits": {
      "primary": {
        "used_percent": 0.0,
        "window_minutes": 300,
        "resets_at": 1769251526
      },
      "secondary": {
        "used_percent": 4.0,
        "window_minutes": 10080,
        "resets_at": 1769392822
      }
    }
  }
}
```

## UI Layout

카드 기반 레이아웃. lipgloss 스타일링.

```
╭─ codex-hud ──────────────────────────────────────────────────╮
│                                                              │
│   ● gpt-5.4          medium          untrusted    v0.111.0   │
│                                                              │
│  ┌─ Context ────────────────────────────────────────────────┐│
│  │  ████████████░░░░░░░░░░░░░░░░░░░░░░░░░░  27.2%          ││
│  │  70,285 / 258,400 tokens                                 ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌─ Tokens ────────────────────┐  ┌─ Session ─────────────┐ │
│  │  ↓ in     1,385,034        │  │  ⏱ 20m 14s   8 turns  │ │
│  │  ↻ cache  1,270,784        │  │  📂 ~/my-project       │ │
│  │  ↑ out       11,636        │  │  🌿 main*  ↑2 ↓0      │ │
│  │  💭 reason    2,548        │  │     +3 ~2 -1           │ │
│  └─────────────────────────────┘  └────────────────────────┘ │
│                                                              │
│  ┌─ Rate Limit ─────────────────────────────────────────────┐│
│  │  ██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   4%           ││
│  │  resets in 2h 13m                                        ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
│  ┌─ Activity ───────────────────────────────────────────────┐│
│  │  ▸ exec_command (running...)                             ││
│  │  exec_command ×55  │  apply_patch ×2                     ││
│  └──────────────────────────────────────────────────────────┘│
│                                                              │
╰──────────────────────────────────────────────────────────────╯
```

### Color Scheme

- Context bar: green (< 50%) → yellow (50-75%) → red (75%+)
- Rate limit bar: green (< 50%) → yellow (50-80%) → red (80%+)
- Active tool: yellow blinking
- Card borders: subtle gray (lipgloss.Color("240"))
- Model name: cyan bold
- Numbers: white bold
- Labels: dim white

### Responsive Behavior

- 터미널 폭 >= 80: 2컬럼 (Tokens | Session)
- 터미널 폭 < 80: 1컬럼 (Tokens 위, Session 아래)
- 터미널 폭 < 60: 미니멀 모드 (핵심 정보만)

## Execution Modes

### Wrapper Mode (기본)

`codex-hud` 실행 시 환경을 자동 감지하여 codex + HUD를 동시에 띄운다.

```
환경 감지 순서:
1. $TMUX 환경변수 → tmux split-pane
2. $WT_SESSION 환경변수 → Windows Terminal wt split-pane
3. $TERM_PROGRAM (iTerm2/WezTerm) → 네이티브 split API
4. fallback → 새 터미널 창 열기 (macOS: osascript, Linux: xdg-terminal, Windows: start)
```

각 세션마다 독립적으로 동작:
- 터미널 1에서 `codex-hud` → codex 세션 A + HUD A
- 터미널 2에서 `codex-hud` → codex 세션 B + HUD B
- 각 HUD는 자기 세션의 .jsonl만 추적

### Watch Mode (독립 실행)

이미 실행 중인 codex 세션을 별도로 모니터링할 때.

```bash
codex-hud --watch              # 가장 최근 세션 자동 감지
codex-hud --watch --file <path> # 특정 세션 파일 지정
```

## CLI Interface

```bash
# wrapper 모드 (기본): codex + HUD 동시 실행
codex-hud

# codex에 인자 전달
codex-hud -- --model gpt-5.3-codex

# watch 모드: 기존 세션 모니터링
codex-hud --watch
codex-hud --watch --file <path>

# 분할 방향 지정 (환경 자동 감지 오버라이드)
codex-hud --split=bottom      # 하단 (기본)
codex-hud --split=right       # 우측

# 설정
codex-hud config              # 인터랙티브 설정
codex-hud config --reset      # 기본값 복원

# 버전
codex-hud --version
```

### Keyboard Shortcuts (TUI)

| Key | Action |
|-----|--------|
| `Tab` | 다중 세션일 때 세션 전환 |
| `q` / `Ctrl+C` | 종료 |
| `r` | 강제 새로고침 |
| `t` | 테마 전환 |
| `m` | 미니멀/풀 모드 토글 |

## Configuration

**File:** `~/.codex/codex-hud.toml`

```toml
[display]
theme = "default"          # default | minimal | neon
refresh_ms = 500
show_rate_limit = true
show_activity = true
show_git = true

[git]
show_dirty = true
show_ahead_behind = true
show_file_stats = true

[tmux]
auto_detect = true
position = "bottom"        # bottom | right
size = 30                  # split %
```

## Multi-Session Behavior

### Wrapper 모드
- 각 `codex-hud` 인스턴스가 자기가 시작한 codex 세션만 추적
- 세션 시작 시 생성되는 .jsonl 파일명(UUID)을 캡처하여 해당 파일만 감시
- 독립적이므로 여러 터미널에서 동시 실행 가능

### Watch 모드
- `~/.codex/sessions/` 하위 전체 디렉토리를 재귀 감시
- 가장 최근에 write가 발생한 `.jsonl` 파일을 자동 추적
- `Tab` 키로 활성 세션 목록 표시 + 전환
- 특정 세션에 고정하려면 `--file` 플래그 사용

## Error Handling

| Condition | Behavior |
|-----------|----------|
| No Codex session | "Waiting for Codex session..." 표시, 폴더 감시 대기 |
| Session ended | "Session ended" 표시, 마지막 상태 유지, 새 세션 자동 전환 |
| JSONL parse error | 해당 줄 스킵, 나머지 계속 처리 |
| rate_limits null | Rate Limit 카드 숨김 |
| No git repo | Git 정보 숨김 |
| Terminal too narrow | 미니멀 모드 자동 전환 |

## Dependencies (Go)

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components (progress bar, etc.) |
| `github.com/fsnotify/fsnotify` | Cross-platform file watching |
| `github.com/BurntSushi/toml` | Config parsing |
| `github.com/spf13/cobra` | CLI flags/subcommands |

## Project Structure

```
codex-hud/
├── cmd/
│   └── codex-hud/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # TOML config loading
│   ├── parser/
│   │   ├── parser.go         # JSONL line parser
│   │   └── types.go          # Event type structs
│   ├── watcher/
│   │   └── watcher.go        # fsnotify + tail logic
│   ├── state/
│   │   └── state.go          # Session state aggregation
│   ├── git/
│   │   └── git.go            # Git status subprocess
│   ├── launcher/
│   │   ├── launcher.go       # Environment detection + split dispatch
│   │   ├── tmux.go           # tmux split-pane
│   │   ├── wt.go             # Windows Terminal split-pane
│   │   └── fallback.go       # New window fallback (osascript, xdg, start)
│   └── tui/
│       ├── model.go           # bubbletea Model
│       ├── update.go          # bubbletea Update
│       ├── view.go            # bubbletea View
│       ├── styles.go          # lipgloss styles
│       └── components/
│           ├── header.go      # Model info header
│           ├── context.go     # Context window card
│           ├── tokens.go      # Token usage card
│           ├── session.go     # Session info card
│           ├── ratelimit.go   # Rate limit card
│           └── activity.go    # Tool activity card
├── docs/
│   └── plans/
│       └── 2026-03-07-codex-hud-design.md
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Cross-Platform Build

```makefile
# Makefile
build-all:
	GOOS=darwin  GOARCH=amd64 go build -o dist/codex-hud-darwin-amd64 ./cmd/codex-hud
	GOOS=darwin  GOARCH=arm64 go build -o dist/codex-hud-darwin-arm64 ./cmd/codex-hud
	GOOS=linux   GOARCH=amd64 go build -o dist/codex-hud-linux-amd64 ./cmd/codex-hud
	GOOS=windows GOARCH=amd64 go build -o dist/codex-hud-windows-amd64.exe ./cmd/codex-hud
```

## Platform-Specific Notes

### Windows
- fsnotify는 Windows에서 ReadDirectoryChangesW 사용 (정상 동작)
- Windows Terminal: `wt split-pane` CLI로 네이티브 split 지원
- PowerShell/Git Bash (비-WT): 새 창 열기 fallback (`start cmd /k codex-hud --watch`)
- 유니코드/이모지: Windows Terminal은 OK, 구형 cmd.exe는 fallback 문자 사용
- Codex 세션 경로: `%USERPROFILE%\.codex\sessions\` (Go의 os.UserHomeDir()로 해결)

### macOS
- tmux 자동 감지 지원
- iTerm2, Warp, WezTerm 전부 호환

### Linux
- tmux 자동 감지 지원
- 대부분의 터미널 에뮬레이터 호환
