@echo off
REM ============================================
REM  DBAS ? Push to GitHub
REM  Run this script to publish the repo
REM ============================================

echo.
echo [1/4] Committing staged files...
git commit -m "DBAS v0.1.0 - Full release"

echo.
echo [2/4] Pushing to GitHub...
git push -u origin main

echo.
echo [3/4] Creating v0.1.0 tag...
git tag v0.1.0

echo.
echo [4/4] Pushing tag...
git push origin v0.1.0

echo.
echo === DONE ===
echo Your repo is live at: https://github.com/Throokie/dbas
echo.
pause
