# codex-hud Windows uninstaller
# Usage: powershell -ExecutionPolicy Bypass -File uninstall.ps1

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\codex-hud"
$BinaryPath = Join-Path $InstallDir "codex-hud.exe"

# Remove binary
if (Test-Path $BinaryPath) {
    Remove-Item -Path $BinaryPath -Force
    Write-Host "Removed $BinaryPath"
} else {
    Write-Host "codex-hud.exe not found at $BinaryPath"
}

# Remove empty install directory
if ((Test-Path $InstallDir) -and ((Get-ChildItem $InstallDir | Measure-Object).Count -eq 0)) {
    Remove-Item -Path $InstallDir -Force
    Write-Host "Removed $InstallDir"
}

# Remove from user PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -like "*$InstallDir*") {
    $NewPath = ($UserPath -split ";" | Where-Object { $_ -ne $InstallDir }) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "Removed $InstallDir from user PATH"
}

Write-Host ""
Write-Host "Done! codex-hud has been uninstalled." -ForegroundColor Green
