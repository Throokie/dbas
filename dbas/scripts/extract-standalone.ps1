# DBAS Standalone Repo Extraction Script
# Run this from the reasonix repo root to extract DBAS into a standalone repo
#
# Usage:
#   powershell -File dbas/scripts/extract-standalone.ps1 -TargetDir ..\dbas-standalone

param(
    [string]$TargetDir = "..\dbas-standalone",
    [string]$ReasonixRoot = (Get-Location).Path
)

$ErrorActionPreference = "Stop"
Write-Host "Extracting DBAS to: $TargetDir"
Write-Host "Reasonix root: $ReasonixRoot"

# 1. Create target directory
New-Item -ItemType Directory -Force -Path $TargetDir | Out-Null
$dbasSource = Join-Path $ReasonixRoot "dbas"

# 2. Copy DBAS files (excluding build artifacts and data)
$excludePatterns = @("dbas-data", "*.exe", "*.test", "coverage.out", "coverage.html", "node_modules")
Get-ChildItem -Path $dbasSource -Recurse -File | Where-Object {
    $exclude = $false
    foreach ($pat in $excludePatterns) {
        if ($_.FullName -like "*$pat*") { $exclude = $true; break }
    }
    -not $exclude
} | ForEach-Object {
    $relPath = $_.FullName.Substring($dbasSource.Length + 1)
    $targetFile = Join-Path $TargetDir $relPath
    $targetParent = Split-Path $targetFile -Parent
    New-Item -ItemType Directory -Force -Path $targetParent | Out-Null
    Copy-Item -Path $_.FullName -Destination $targetFile
}

Write-Host "Step 1/3: DBAS files copied"

# 3. Copy minimal reasonix interfaces needed by DBAS
$neededPackages = @(
    "internal/provider",
    "internal/tool",
    "internal/agent",
    "internal/event",
    "internal/nilutil"
)

foreach ($pkg in $neededPackages) {
    $srcDir = Join-Path $ReasonixRoot $pkg
    $tgtDir = Join-Path $TargetDir "reasonix" $pkg
    if (Test-Path $srcDir) {
        New-Item -ItemType Directory -Force -Path $tgtDir | Out-Null
        Get-ChildItem -Path $srcDir -Filter "*.go" | ForEach-Object {
            Copy-Item -Path $_.FullName -Destination (Join-Path $tgtDir $_.Name)
        }
        Write-Host "  Copied: reasonix/$pkg"
    }
}

# Also copy the openai provider and builtin tools
$extraPackages = @(
    "internal/provider/openai",
    "internal/tool/builtin"
)
foreach ($pkg in $extraPackages) {
    $srcDir = Join-Path $ReasonixRoot $pkg
    $tgtDir = Join-Path $TargetDir "reasonix" $pkg
    if (Test-Path $srcDir) {
        New-Item -ItemType Directory -Force -Path $tgtDir | Out-Null
        Get-ChildItem -Path $srcDir -Filter "*.go" | ForEach-Object {
            Copy-Item -Path $_.FullName -Destination (Join-Path $tgtDir $_.Name)
        }
        Write-Host "  Copied: reasonix/$pkg"
    }
}

Write-Host "Step 2/3: Reasonix interfaces copied"

# 4. Create standalone go.mod
$goMod = @'
module github.com/dbas-ai/dbas

go 1.25.0

toolchain go1.26.4

require (
    github.com/BurntSushi/toml v1.6.0
)

replace reasonix => ./reasonix
'@

Set-Content -Path (Join-Path $TargetDir "go.mod") -Value $goMod -Encoding UTF8

Write-Host "Step 3/3: go.mod created"
Write-Host ""
Write-Host "Done! DBAS extracted to: $TargetDir"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  cd $TargetDir"
Write-Host "  go mod tidy"
Write-Host "  go build ./cmd/dbas"
Write-Host "  ./dbas --demo --port 8080"
Write-Host ""
Write-Host "To publish as a standalone repo:"
Write-Host "  cd $TargetDir"
Write-Host "  git init"
Write-Host "  git add ."
Write-Host "  git commit -m 'Initial standalone DBAS extraction'"
Write-Host "  git remote add origin https://github.com/YOUR_USER/dbas.git"
Write-Host "  git push -u origin main"
