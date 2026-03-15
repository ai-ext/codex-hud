# 변경 이력

## [0.4.0] - 2026-03-15

### 새로운 기능
- **실시간 스킬 표시** — Codex 세션에서 호출된 스킬(`$skill-name`) 실시간 감지 및 표시

### 버그 수정
- **Windows Terminal 분할** — 같은 창에서 codex(위) + HUD(아래) 분할 구성
- **Windows 세션 감지** — fsnotify 누락 시 300ms 폴링 백업으로 실시간 감지
- **Windows 실시간 업데이트** — 200ms 폴링으로 파일 이벤트 누락 보완
- **카드 텍스트 잘림** — lipgloss 테두리/패딩 고려한 너비 계산 수정
- **fresh 모드 데이터 오염** — 이전 세션 데이터가 로드되는 문제 해결

## [0.3.3] - 2026-03-15

### 버그 수정
- **Windows 세션 감지 실패** — fsnotify가 Windows에서 디렉토리/파일 이벤트를 놓치는 문제. 폴링 백업 추가로 최대 1초 내 새 세션 감지
- **Windows 실시간 업데이트** — 파일 Write 이벤트 누락 시에도 500ms 간격 폴링으로 실시간 반영

## [0.3.2] - 2026-03-15

### 버그 수정
- **Windows 분할 창 문제** — codex가 원래 터미널에서 실행되고 새 WT 창에 빈 셸이 뜨던 문제. `wt new-tab codex ; split-pane ... codex-hud` 방식으로 통일하여 같은 창에서 codex(위) + HUD(아래) 함께 구성

## [0.3.1] - 2026-03-15

### 버그 수정
- **Windows Terminal 분할 방향** — WT 플래그 매핑이 반대로 적용되어 좌/우 분할되던 문제. 맥과 동일하게 위/아래 분할로 수정
- **WT 외부 실행 지원** — PowerShell/CMD/Git Bash에서 실행 시 빈 셸 + HUD만 표시되던 문제. 새 WT 창에 codex + HUD를 함께 분할 구성하여 시작

## [0.3.0] - 2026-03-15

### 새로운 기능
- **실시간 스킬 표시** — Codex 세션에서 호출된 스킬(`$skill-name`)을 실시간으로 감지하여 HUD에 표시
- **스킬 레지스트리 자동 파싱** — `turn_context`의 `user_instructions`에서 등록된 스킬 목록을 자동으로 파싱
- **턴 단위 스킬 초기화** — 매 턴마다 활성 스킬 목록을 초기화하여 현재 턴에서 사용 중인 스킬만 표시

### 버그 수정
- **카드 내부 텍스트 잘림** — Context, Usage 카드에서 내용이 우측으로 넘치던 문제. lipgloss 테두리/패딩을 고려한 너비 계산 수정
- **세션 감지 레이스 컨디션** — watcher 설정과 이벤트 루프 시작 사이에 생성된 세션 파일을 놓치던 문제. 디렉토리 스캔으로 보완

## [0.2.3] - 2026-03-14

### 변경 사항
- **Windows 새 창 폴백 제거** — WT 없을 때 별도 창 열기 대신, Windows Terminal 설치를 안내/자동 설치 후 종료
- **Windows Terminal 필수** — Windows에서 split-pane 경험을 보장하기 위해 WT가 없으면 `winget`으로 설치 유도
- **install.ps1 개선** — WT 미설치 시 winget 자동 설치 + 릴리즈 다운로드 안내 추가

## [0.2.2] - 2026-03-14

### 버그 수정
- **Windows 플랫폼 오류** — Windows에서 tmux 설치를 요구하던 문제 수정. Windows Terminal/새 창 열기/watch 모드 순으로 자동 폴백
- **wt split-pane 인자 처리** — 바이너리 경로와 플래그를 개별 인자로 분리하여 "파일 찾을 수 없음" 오류 해결
- **Git Bash 호환성** — Windows에서 `os.Executable()` 경로의 심볼릭 링크 해석 및 경로 정규화
- **codex.cmd 감지** — npm으로 설치된 codex (`codex.cmd`)를 Windows에서 자동 감지

### 새로운 기능
- **Windows Terminal 자동 설치** — `install.ps1` 실행 시 Windows Terminal 미설치면 `winget`으로 자동 설치
- **Windows 새 창 폴백** — Windows Terminal 없어도 `cmd /c start`로 별도 창에 HUD 실행
- **wt.exe PATH 감지** — `WT_SESSION` 환경변수 없어도 PATH에서 `wt.exe` 탐지

### 변경 사항
- Windows에서 tmux 탐색 제거 (macOS/Linux에서만 시도)
- Windows fallback 안내 메시지를 Windows Terminal 안내로 변경

## [0.2.1] - 2026-03-14

### 새로운 기능
- **적응형 레이아웃** — 터미널 높이에 따라 자동 전환: 카드 레이아웃 (넓을 때) ↔ 콤팩트 모드 (좁을 때)
- **`--size` 옵션** — 분할 패널 크기를 퍼센트로 지정 (예: `--size=35`, 기본값 40%)
- **Waiting 상태 Usage 표시** — 세션 연결 전에도 WHAM API 사용량(Rate Limits) 즉시 표시
- **뷰포트 클리핑** — 콘텐츠가 터미널보다 클 때 상단(헤더)을 항상 유지하고 하단을 자름
- **패널 높이 꽉 채우기** — 외곽 테두리가 항상 터미널 높이 전체를 채움

### 버그 수정
- **헤더 잘림** — 분할 패널에서 모델명/버전 등 상단 정보가 잘리던 문제. 적응형 레이아웃 + 뷰포트 클리핑으로 수정

### 변경 사항
- 기본 분할 패널 크기 30% → 40%로 증가
- 콤팩트 모드: Context, Tokens, Session, Usage, Activity를 각 1-2줄로 압축

## [0.2.0] - 2026-03-13

### 새로운 기능
- **실시간 사용량 (Rate Limits)** — WHAM API를 통한 5시간/7일 사용량 실시간 표시, Codex `/status`와 동일
- **자동 세션 전환** — 새로운 Codex 세션을 자동 감지하여 전환
- **tmux 자동 실행** — tmux가 설치되어 있지만 세션 밖일 때, 분할 패인으로 새 세션 생성
- **Windows Terminal 지원** — `wt split-pane`으로 네이티브 Windows 분할 지원
- **사전 로딩** — TUI 시작 전 기존 세션 데이터를 즉시 로드 (시작 시 깜빡임 없음)
- **Windows 설치 스크립트** — `install.bat` / `install.ps1`으로 원클릭 PATH 설정
- **`make install` / `make uninstall`** — `/usr/local/bin`에 빠른 설치/삭제

### 버그 수정
- **새 세션에서 컨텍스트 잔류** — 새 세션 시작 시 이전 세션의 45% 컨텍스트가 표시되던 문제. 세션 ID 변경 시 세션별 상태를 초기화하여 수정
- **사용량 깜빡임** — 시작 시 이전 Rate Limit 데이터(1%/4%)가 잠깐 표시되던 문제. 세션 파일의 Rate Limit 폴백을 제거하고, WHAM API 응답 후에만 Usage 카드 표시
- **TailFile 중복** — 사전 로딩된 줄이 채널로 다시 처리되던 문제. 파일 끝에서 시작하는 `TailFileFromEnd()`로 수정
- **래퍼 모드 미작동** — macOS에서 `codex-hud`가 HUD 없이 codex만 실행하던 문제. tmux 자동 감지 및 분할 세션 생성으로 수정
- **기본 모드 혼동** — 래퍼 모드(codex + HUD 함께)를 기본값으로 복원; `--watch`는 HUD 전용
- **Rate Limits null 처리** — 최신 Codex (v0.114.0+)에서 `rate_limits: null`을 보내 기존 Rate Limit 데이터가 지워지던 문제 수정
- **컨텍스트 토큰 표시** — Codex `/status` 출력과 일치하도록 원시 토큰 값 표시 (예: `8,590 / 258,400`)

### 변경 사항
- Rate Limits는 이제 WHAM API에서만 조회 (더 이상 세션 파일 데이터 사용 안 함)
- `--fresh` 플래그 추가 (래퍼 모드에서 이전 세션 사전 로딩을 건너뛸 때 내부적으로 사용)
- 폴백 런처 단순화 — OS별 터미널 실행 시도 대신 안내 메시지 출력

## [0.1.0] - 2026-03-12

### 최초 릴리즈
- Codex JSONL 파일에서 실시간 세션 모니터링
- 컨텍스트 윈도우 사용량 프로그레스 바
- 토큰 통계 (입력, 캐시, 출력, 추론)
- 세션 정보 (경과 시간, 턴 수, 작업 디렉토리)
- 활동 추적 (실행 중/완료된 도구 호출)
- Git 상태 표시
- `~/.codex/codex-hud.toml`로 설정 가능
- 크로스 플랫폼 빌드 (macOS, Linux, Windows)
- tmux 분할 패인 연동
- 최신 세션 파일 자동 감지
