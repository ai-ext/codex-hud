.PHONY: build run test clean build-all install uninstall

build:
	go build -o dist/codex-hud ./cmd/codex-hud

run:
	go run ./cmd/codex-hud

test:
	go test ./... -v

clean:
	rm -rf dist/

install: build
	cp dist/codex-hud /usr/local/bin/codex-hud
	@echo "Installed to /usr/local/bin/codex-hud"

uninstall:
	rm -f /usr/local/bin/codex-hud
	@echo "Removed /usr/local/bin/codex-hud"

build-all:
	GOOS=darwin  GOARCH=amd64 go build -o dist/codex-hud-darwin-amd64 ./cmd/codex-hud
	GOOS=darwin  GOARCH=arm64 go build -o dist/codex-hud-darwin-arm64 ./cmd/codex-hud
	GOOS=linux   GOARCH=amd64 go build -o dist/codex-hud-linux-amd64 ./cmd/codex-hud
	GOOS=windows GOARCH=amd64 go build -o dist/codex-hud-windows-amd64.exe ./cmd/codex-hud
	@echo ""
	@echo "Windows install: copy dist/codex-hud-windows-amd64.exe to the same"
	@echo "directory as scripts/install.ps1, then run:"
	@echo "  powershell -ExecutionPolicy Bypass -File scripts/install.ps1"
