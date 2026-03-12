# codex-hud Windows installer
# Usage: powershell -ExecutionPolicy Bypass -File install.ps1
#
# This script:
#   1. Downloads or copies the codex-hud binary
#   2. Places it in a persistent directory
#   3. Adds that directory to the user's PATH

$ErrorActionPreference = "Stop"

$InstallDir = "$env:LOCALAPPDATA\codex-hud"
$BinaryName = "codex-hud.exe"
$BinaryPath = Join-Path $InstallDir $BinaryName

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "Created $InstallDir"
}

# Find the binary - check common locations
$Source = $null
$SearchPaths = @(
    ".\codex-hud-windows-amd64.exe",
    ".\codex-hud.exe",
    ".\dist\codex-hud-windows-amd64.exe",
    ".\dist\codex-hud.exe"
)

foreach ($p in $SearchPaths) {
    if (Test-Path $p) {
        $Source = $p
        break
    }
}

if (-not $Source) {
    Write-Host "Error: codex-hud binary not found." -ForegroundColor Red
    Write-Host ""
    Write-Host "Place one of these files in the current directory:"
    Write-Host "  - codex-hud-windows-amd64.exe"
    Write-Host "  - codex-hud.exe"
    Write-Host ""
    Write-Host "Or build from source:"
    Write-Host "  go build -o codex-hud.exe ./cmd/codex-hud"
    exit 1
}

# Copy binary
Copy-Item -Path $Source -Destination $BinaryPath -Force
Write-Host "Installed $Source -> $BinaryPath"

# Add to user PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to user PATH"
    Write-Host ""
    Write-Host "** Restart your terminal for PATH changes to take effect **" -ForegroundColor Yellow
} else {
    Write-Host "$InstallDir is already in PATH"
}

Write-Host ""
Write-Host "Done! Run 'codex-hud' from any directory." -ForegroundColor Green
