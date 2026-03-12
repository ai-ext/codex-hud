# codex-hud

[OpenAI Codex CLI](https://github.com/openai/codex)용 실시간 모니터링 HUD. Codex 옆에 분할 패널로 실행되어 세션 상태를 실시간으로 보여줍니다.

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
│  │ 5h  █████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 12%        │   │
│  │ 7d  ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 10%        │   │
│  ╰──────────────────────────────────────────────────────────╯   │
│  ╭──────────────────────────────────────────────────────────╮   │
│  │ Activity                                                  │   │
│  │ ▶ exec_command                                            │   │
│  │ exec_command x53                                          │   │
│  ╰──────────────────────────────────────────────────────────╯   │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

## 기능

- **컨텍스트 윈도우** — 토큰 사용량 실시간 프로그레스 바
- **토큰 통계** — 입력, 캐시, 출력, 추론 토큰
- **사용량 (Rate Limits)** — WHAM API 기반 5시간/7일 사용량
- **세션 정보** — 경과 시간, 턴 수, 작업 디렉토리
- **활동 내역** — 실행 중/완료된 도구 호출
- **Git 상태** — 브랜치, 변경 사항 (선택)
- **자동 세션 전환** — 새 Codex 세션 자동 감지

## 설치

### 바이너리 다운로드 (Go 불필요)

[GitHub Releases](https://github.com/ai-ext/codex-hud/releases)에서 OS에 맞는 바이너리를 다운로드하세요.

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
자동으로 `%LOCALAPPDATA%\codex-hud\`에 복사하고 PATH에 추가됩니다.

### 소스에서 빌드 (Go 1.21+ 필요)

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

# 모든 플랫폼 바이너리 빌드
make build-all
```

## 사용법

### 기본 모드 (래퍼 모드)

```bash
codex-hud
```

Codex + HUD를 분할 패널로 함께 실행합니다. 자동 감지:
- **tmux** — 현재 패인을 분할 (또는 새 세션 생성)
- **Windows Terminal** — `wt split-pane` 사용
- **기타** — HUD 전용 감시 모드로 폴백

### 감시 모드 (HUD만)

```bash
codex-hud --watch
```

HUD 패널만 실행합니다. 다른 터미널에서 Codex를 별도로 실행하세요. 이미 Codex가 실행 중일 때 유용합니다.

### 옵션

```
--watch          HUD 패널만 실행 (Codex는 별도 실행)
--file <path>    특정 .jsonl 세션 파일 모니터링
--split bottom   분할 방향: bottom (기본값) 또는 right
--version        버전 표시
```

## 설정

`~/.codex/codex-hud.toml` (선택사항):

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

## 동작 원리

1. Codex CLI가 세션 이벤트를 `~/.codex/sessions/*.jsonl`에 기록
2. codex-hud가 fsnotify로 파일 변경을 실시간 감시
3. `token_count` 이벤트에서 컨텍스트 사용량, 토큰 통계 추출
4. WHAM API에서 Rate Limit 조회 (Codex `/status`와 동일)
5. [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI로 렌더링

## 단축키

| 키 | 동작 |
|-----|--------|
| `q` / `Ctrl+C` | 종료 |
| `r` | Git 상태 새로고침 |

## 요구사항

- [OpenAI Codex CLI](https://github.com/openai/codex) 설치 및 인증 완료
- Go 1.21+ (소스에서 빌드할 경우에만)

## 삭제

```bash
# macOS/Linux
make uninstall

# Windows
powershell -ExecutionPolicy Bypass -File scripts/uninstall.ps1
```

## 라이선스

MIT
