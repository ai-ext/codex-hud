.PHONY: build run test clean build-all

build:
	go build -o dist/codex-hud ./cmd/codex-hud

run:
	go run ./cmd/codex-hud

test:
	go test ./... -v

clean:
	rm -rf dist/

build-all:
	GOOS=darwin  GOARCH=amd64 go build -o dist/codex-hud-darwin-amd64 ./cmd/codex-hud
	GOOS=darwin  GOARCH=arm64 go build -o dist/codex-hud-darwin-arm64 ./cmd/codex-hud
	GOOS=linux   GOARCH=amd64 go build -o dist/codex-hud-linux-amd64 ./cmd/codex-hud
	GOOS=windows GOARCH=amd64 go build -o dist/codex-hud-windows-amd64.exe ./cmd/codex-hud
