# DBAS Build And Run

Version: v0.1.0-alpha

DBAS is the disciplined multi-branch control plane for Reasonix. It owns branch lifecycle, FSM governance, dual-brain review, verification, persistence, and kanban interaction. Execution should be delegated to mature backends where possible: Reasonix native tools by default, or command executors such as mini-swe-agent and aider.

## Prerequisites

| Dependency | Notes |
| --- | --- |
| Go | Use the Go version required by the parent Reasonix module. |
| Git | Needed by executor backends and normal code workflows. |
| Model endpoint | Any OpenAI-compatible endpoint configured in `dbas.toml`. |

Secrets are never stored in source. If an API key was previously committed or shared, rotate it before using this project again.

## Configure

```powershell
Copy-Item dbas.example.toml dbas.toml
notepad dbas.toml
```

For cloud providers, set only the env var named by `api_key_env`:

```powershell
$env:DEEPSEEK_API_KEY = "..."
```

Local/private OpenAI-compatible endpoints can omit `api_key_env`.

## Build

```powershell
go build -o dbas.exe ./cmd/dbas
```

## Run

```powershell
.\dbas.exe -config .\dbas.toml
```

Useful flags override config values:

```powershell
.\dbas.exe -config .\dbas.toml -port 8081 -root .\dbas-data -max-parallel 6
```

## Executor Backends

Default native backend:

```toml
[executor]
type = "reasonix-native"
```

Command backend for existing agents:

```toml
[executor]
type = "command"
command = "mini"
args = ["--model", "openai/deepseek-chat", "--workdir", "{workspace}", "--"]
workdir = "{workspace}"
stdin_format = "text"
```

This is the intended growth path: DBAS coordinates and verifies; mini-swe-agent, aider, OpenHands-compatible agents, or other mature tools execute.

Command executors receive `DBAS_WORKSPACE_DIR`, `DBAS_REPORT_FILE`, `DBAS_LOG_FILE`, and `DBAS_TASK_FILE`. They should write changes into the workspace and produce the report file. `stdin_format = "json"` sends the full task object for wrapper scripts; `text` sends human-readable instructions for CLIs. `args`, `env`, and `workdir` can use `{workspace}`, `{report}`, `{log}`, `{task_file}`, `{branch_id}`, and `{attempt}` placeholders.

## Verify

```powershell
go test ./...
```

Health check after startup:

```powershell
Invoke-RestMethod http://localhost:8080/health
```
