# 변경 이력

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
