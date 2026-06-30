@echo off
REM DBAS Quick Start — try DBAS in 30 seconds
REM No API keys, no Docker, no configuration needed.

echo.
echo   DBAS Quick Start
echo   ================
echo.

REM Check Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo   ERROR: Go is not installed.
    echo   Install Go 1.25+: https://go.dev/dl/
    exit /b 1
)

go version
echo   ✓ Go detected

REM Navigate to repo root (parent of dbas/)
cd /d "%~dp0..\.."

echo   ✓ Building DBAS...

go build -o dbas.exe ./dbas/cmd/dbas
if %ERRORLEVEL% NEQ 0 (
    echo   ERROR: Build failed.
    exit /b 1
)

echo   ✓ Build complete
echo.
echo   Starting DBAS in DEMO mode...
echo   Dashboard: http://localhost:8080
echo.
dbas.exe --demo --port 8080

