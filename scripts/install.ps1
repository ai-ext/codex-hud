# codex-hud Windows installer
# Usage: powershell -ExecutionPolicy Bypass -File install.ps1
#
# This script:
#   1. Ensures Windows Terminal is installed (for split-pane support)
#   2. Downloads or copies the codex-hud binary
#   3. Places it in a persistent directory
#   4. Adds that directory to the user's PATH

$ErrorActionPreference = "Stop"

# ── Step 1: Check / install Windows Terminal ──────────────────────────
Write-Host "Checking Windows Terminal..." -ForegroundColor Cyan

$wtInstalled = $false

# Check if wt.exe is in PATH
if (Get-Command "wt" -ErrorAction SilentlyContinue) {
    $wtInstalled = $true
}

# Also check the default install location (Microsoft Store apps)
if (-not $wtInstalled) {
    $wtAppPath = "$env:LOCALAPPDATA\Microsoft\WindowsApps\wt.exe"
    if (Test-Path $wtAppPath) {
        $wtInstalled = $true
    }
}

if ($wtInstalled) {
    Write-Host "  Windows Terminal found." -ForegroundColor Green
} else {
    Write-Host "  Windows Terminal not found." -ForegroundColor Yellow
    Write-Host "  Windows Terminal is needed for split-pane HUD display." -ForegroundColor Yellow
    Write-Host ""

    # Try winget first
    if (Get-Command "winget" -ErrorAction SilentlyContinue) {
        Write-Host "  Installing Windows Terminal via winget..." -ForegroundColor Cyan
        try {
            winget install --id Microsoft.WindowsTerminal --accept-source-agreements --accept-package-agreements
            Write-Host "  Windows Terminal installed successfully." -ForegroundColor Green
        } catch {
            Write-Host "  Failed to install Windows Terminal automatically." -ForegroundColor Red
            Write-Host "  Please install manually: https://aka.ms/terminal" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  winget not available. Please install Windows Terminal manually:" -ForegroundColor Yellow
        Write-Host "    https://aka.ms/terminal" -ForegroundColor White
        Write-Host "    or: Microsoft Store -> 'Windows Terminal'" -ForegroundColor White
    }
    Write-Host ""
}

# ── Step 2: Install codex-hud binary ─────────────────────────────────
$InstallDir = "$env:LOCALAPPDATA\codex-hud"
$BinaryName = "codex-hud.exe"
$BinaryPath = Join-Path $InstallDir $BinaryName
$DownloadUrl = "https://github.com/ai-ext/codex-hud/releases/latest/download/codex-hud-windows-amd64.exe"

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Host "Created $InstallDir"
}

# Find the binary locally first, otherwise download from GitHub
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

if ($Source) {
    Write-Host "Found local binary: $Source" -ForegroundColor Green
    Copy-Item -Path $Source -Destination $BinaryPath -Force
} else {
    Write-Host "Downloading codex-hud from GitHub..." -ForegroundColor Cyan
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $BinaryPath -UseBasicParsing
        Write-Host "  Downloaded successfully." -ForegroundColor Green
    } catch {
        Write-Host "Error: Failed to download codex-hud." -ForegroundColor Red
        Write-Host "  URL: $DownloadUrl"
        Write-Host "  $($_.Exception.Message)"
        Write-Host ""
        Write-Host "Download manually from: https://github.com/ai-ext/codex-hud/releases/latest"
        exit 1
    }
}

# Remove "downloaded from internet" block so Windows doesn't flag it
Unblock-File -Path $BinaryPath -ErrorAction SilentlyContinue
Write-Host "Installed to $BinaryPath"

# ── Step 3: Add to user PATH ─────────────────────────────────────────
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
Write-Host "Done! Run 'codex-hud' to start." -ForegroundColor Green
Write-Host "  codex-hud          -> Codex + HUD split pane" -ForegroundColor White
Write-Host "  codex-hud --watch  -> HUD only (monitor mode)" -ForegroundColor White
