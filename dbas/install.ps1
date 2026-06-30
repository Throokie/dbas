# DBAS Installer for Windows
# Run: irm https://git.io/dbas.ps1 | iex

$ErrorActionPreference = 'Stop'
$tmp = Join-Path $env:TEMP "dbas-install"
if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
git clone --depth=1 https://github.com/esengine/DeepSeek-Reasonix.git $tmp 2>&1 | Out-Null
Push-Location $tmp
go build -ldflags="-s -w" -o "$env:USERPROFILE\.dbas\dbas.exe" .\dbas\cmd\dbas
$dir = "$env:USERPROFILE\.dbas"
$p = [Environment]::GetEnvironmentVariable("Path","User")
if ($p -notlike "*$dir*") { [Environment]::SetEnvironmentVariable("Path","$p;$dir","User") }
Pop-Location
Remove-Item -Recurse -Force $tmp
Write-Host "DBAS installed to $dir\dbas.exe" -ForegroundColor Green
Write-Host ""
Write-Host "Run: dbas --demo --port 8080" -ForegroundColor Cyan
