# DBAS Demo Script
# Run this to see a guided tour of DBAS features
#
# Usage:
#   powershell -File dbas/scripts/demo-tour.ps1

param(
    [string]$Port = "8080",
    [switch]$NoWait
)

$BASE = "http://localhost:$Port"
$DEMO_BRANCH = "demo-tour-" + (Get-Random -Maximum 99999)

function Step($title, $command) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  $title" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
    if (-not $NoWait) { Start-Sleep -Seconds 2 }
    Invoke-Expression $command
}

# Step 1: Health check
Step "Health Check" {
    $r = Invoke-RestMethod -Uri "$BASE/health"
    Write-Host "Status: $($r.status)" -ForegroundColor Green
    Write-Host "Timestamp: $($r.timestamp)" -ForegroundColor Gray
}

# Step 2: List available skills
Step "Available Skills" {
    $r = Invoke-RestMethod -Uri "$BASE/api/dbas/skills"
    Write-Host "Loaded $($r.count) skills:" -ForegroundColor Green
    foreach ($s in $r.skills) {
        Write-Host "  - $($s.name) v$($s.version): $($s.description)" -ForegroundColor Yellow
    }
}

# Step 3: List registered plugins
Step "Registered Plugins" {
    $r = Invoke-RestMethod -Uri "$BASE/api/dbas/plugins"
    Write-Host "Executors:" -ForegroundColor Green
    foreach ($e in $r.executors) { Write-Host "  - $($e.name): $($e.description)" -ForegroundColor Yellow }
    Write-Host "Verifiers:" -ForegroundColor Green
    foreach ($v in $r.verifiers) { Write-Host "  - $($v.name): $($v.description)" -ForegroundColor Yellow }
}

# Step 4: View current state
Step "Current Branch State" {
    $r = Invoke-RestMethod -Uri "$BASE/api/dbas/state"
    $count = @($r.branches).Count
    Write-Host "Active branches: $count" -ForegroundColor Green
    Write-Host "Experience slot rules:" -ForegroundColor Green
    $slot = $r.experience_slot
    Write-Host "  Default flow: $($slot.default_flow -join ' -> ')" -ForegroundColor Yellow
    Write-Host "  Escalation threshold: $($slot.rules.escalation_threshold)" -ForegroundColor Yellow
    Write-Host "  Max retries (bug_loop): $($slot.workflows.bug_loop.max_retries)" -ForegroundColor Yellow
}

# Step 5: Spawn a branch
Step "Spawning a Branch (Happy Path)" {
    $body = @{taskName = $DEMO_BRANCH; input = "Implement a user authentication endpoint"} | ConvertTo-Json
    $r = Invoke-RestMethod -Method Post -Uri "$BASE/api/dbas/spawn" -Body $body -ContentType "application/json"
    Write-Host "Branch created: $($r.branch_id)" -ForegroundColor Green
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Demo Complete!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Dashboard: $BASE" -ForegroundColor White
Write-Host "API:       $BASE/api/dbas/state" -ForegroundColor White
Write-Host "Skills:    $BASE/api/dbas/skills" -ForegroundColor White
Write-Host "Plugins:   $BASE/api/dbas/plugins" -ForegroundColor White
