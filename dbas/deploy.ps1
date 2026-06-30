# DBAS Deployment Script (PowerShell)
# Usage: .\deploy.ps1

$ErrorActionPreference = "Stop"

$script:BaseDir = $PSScriptRoot
if (-not $BaseDir) { $BaseDir = (Get-Location).Path }

function Write-Header($text) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  $text" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Step($text) {
    Write-Host "[STEP] $text" -ForegroundColor Yellow
}

function Write-OK($text) {
    Write-Host "  OK  $text" -ForegroundColor Green
}

function Write-Fail($text) {
    Write-Host "  FAIL $text" -ForegroundColor Red
}

function Write-Info($text) {
    Write-Host "  -> $text" -ForegroundColor Gray
}

# =====================
# 1. Check Go Environment
# =====================
Write-Header "1. Check Go Environment"

$goVersion = $null
try {
    $goVersion = & go version 2>$null
} catch {}

if (-not $goVersion) {
    Write-Fail "Go not found. Please install Go 1.25+"
    Write-Info "  winget install GoLang.Go"
    Write-Info "  or visit https://go.dev/dl/"
    exit 1
}

Write-OK "Go installed: $goVersion"

$env:GOTOOLCHAIN = "local"
Write-Info "GOTOOLCHAIN=local (use local Go version)"

# =====================
# 2. Check Project Structure
# =====================
Write-Header "2. Check Project Structure"

$requiredPaths = @(
    (Join-Path $BaseDir "go.mod"),
    (Join-Path $BaseDir "cmd\dbas\main.go"),
    (Join-Path $BaseDir "frontend\dist\index.html")
)

$allExist = $true
foreach ($p in $requiredPaths) {
    if (Test-Path $p) {
        Write-OK (Split-Path $p -Leaf)
    } else {
        Write-Fail (Split-Path $p -Leaf)
        $allExist = $false
    }
}

if (-not $allExist) {
    Write-Fail "Project structure incomplete. Ensure dbas is in the correct directory."
    exit 1
}

# =====================
# 3. Build
# =====================
Write-Header "3. Build DBAS"
Write-Step "Building..."

Push-Location $BaseDir

try {
    if (Test-Path (Join-Path $BaseDir "dbas.exe")) {
        Remove-Item (Join-Path $BaseDir "dbas.exe") -Force
        Write-Info "Cleaned old build"
    }

    $buildOutput = & go build -o (Join-Path $BaseDir "dbas.exe") ./cmd/dbas 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Build failed"
        Write-Host ""
        Write-Host "Build errors:" -ForegroundColor Red
        Write-Host $buildOutput -ForegroundColor Red
        Write-Host ""
        Write-Info "If errors involve 'reasonix/internal/...', ensure the reasonix module is accessible."
        exit 1
    }

    Write-OK "Build successful"
    $exePath = Join-Path $BaseDir "dbas.exe"
    Write-Info "Output: $exePath"
} finally {
    Pop-Location
}

# =====================
# 4. Verify Build
# =====================
Write-Header "4. Verify Build"

$exePath = Join-Path $BaseDir "dbas.exe"
if (Test-Path $exePath) {
    $fileInfo = Get-Item $exePath
    Write-OK "dbas.exe exists"
    Write-Info "Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB"
    Write-Info "Modified: $($fileInfo.LastWriteTime)"
} else {
    Write-Fail "Build artifact not found"
    exit 1
}

# =====================
# 5. Start Service
# =====================
Write-Header "5. Start DBAS"
Write-Step "Starting DBAS..."
Write-Info "Config: .\dbas.toml"
Write-Info "API:    http://localhost:8080"
Write-Info "Kanban: http://localhost:8080"
Write-Info ""
Write-Info "Press Ctrl+C to stop"
Write-Info ""

Push-Location $BaseDir
& .\dbas.exe -config .\dbas.toml
Pop-Location
