@echo off
REM codex-hud Windows uninstaller
REM Usage: just double-click or run: uninstall.bat
powershell -ExecutionPolicy Bypass -File "%~dp0uninstall.ps1"
pause
