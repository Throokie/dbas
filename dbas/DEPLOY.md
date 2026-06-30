# DBAS Deployment Guide

> Version: v0.1.0-alpha

## Prerequisites

- **Go**: Use the version required by the parent Reasonix module.
- **Git**: For cloning and normal development workflows.
- **Model endpoint**: Any OpenAI-compatible endpoint.

Secrets are never stored in source. Cloud providers require environment variables.

## Quick Deploy

### 1. Configure

Copy the example configuration and edit it for your endpoints:

```powershell
Copy-Item dbas.example.toml dbas.toml
notepad dbas.toml
```

For cloud providers, set only the environment variable named by `api_key_env`:

```powershell
$env:DEEPSEEK_API_KEY = "your-key-here"
```

Local/private OpenAI-compatible endpoints can omit `api_key_env`.

### 2. Build

```powershell
go build -o dbas.exe ./cmd/dbas
```

### 3. Run

```powershell
.\dbas.exe -config .\dbas.toml
```

The server starts on the port configured in `dbas.toml` (default 8080).
Open http://localhost:8080 to view the kanban dashboard.

## Troubleshooting

### Error: `go: no go.mod file found`

**Cause**: Running `go build` outside the dbas directory.
**Fix**: Ensure you are in the dbas directory and run `go build -o dbas.exe ./cmd/dbas`.

### Error: `go: errors parsing go.mod: ... toolchain not available`

**Cause**: The parent reasonix module declares a toolchain version newer than your installed Go.
**Fix**: Set `$env:GOTOOLCHAIN = "local"` before building.

### Error: `cannot find package "reasonix/internal/..."`

**Cause**: The reasonix parent module is not fully cloned or is in a different location.
**Fix**: Ensure the reasonix repository is cloned as a sibling or parent of the dbas directory, and that the Go module replacement paths are correct.

### Error: API key authentication failures

**Cause**: Missing or incorrect API key environment variable.
**Fix**: Check the `api_key_env` field in your provider config and ensure the corresponding environment variable is set.

## Verification

After startup, check health:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Spawn a test branch:

```powershell
$body = @{ taskName = "hello-world"; input = "Write a hello world program" } | ConvertTo-Json -Depth 3
Invoke-RestMethod -Uri "http://localhost:8080/api/dbas/spawn" -Method POST -ContentType "application/json" -Body $body
```

---

*DBAS: Disciplined Branching Agent System*  
*Control plane for coding agents. Execution delegated to mature backends.*
