# DBAS Startup Guide

Version: v0.1.0-alpha

DBAS is a disciplined branching agent system built on Reasonix. Its core intent is not to rebuild every coding-agent capability. DBAS should be the control plane: multi-branch orchestration, FSM-enforced lifecycle, dual-brain decision/review, independent verification, kanban/IM interaction, and durable evidence.

The execution layer is pluggable. Start with `reasonix-native`; bridge to `mini-swe-agent`, `aider`, or other proven executors when they fit the task better.

---

## Quick Start

```bash
# 1. Set your API key (supports any OpenAI-compatible provider)
export OPENAI_API_KEY="sk-..."

# 2. Configure
cp dbas.example.toml dbas.toml

# 3. Build
go build -o dbas.exe ./cmd/dbas

# 4. Run (with optional demo mode)
./dbas.exe -config dbas.toml --demo
```

On Windows:

```powershell
Copy-Item dbas.example.toml dbas.toml
$env:OPENAI_API_KEY = "sk-..."
go build -o dbas.exe ./cmd/dbas
.\dbas.exe -config .\dbas.toml --demo
```

### What the demo does

With `--demo`, DBAS auto-spawns a branch that walks through the full FSM lifecycle:
1. INIT → PLAN (Decider analyzes a task)
2. PLAN → EXECUTING (Reviewer approves the plan)
3. EXECUTING → VERIFYING (Executor completes work)
4. VERIFYING → PASS (Verifier checks pass)
5. PASS → ARCHIVE (Evidence stored, branch finalized)

Watch the lifecycle unfold in real-time on the dashboard.

---

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `dbas.toml` | Path to TOML configuration file |
| `-port` | `8080` | HTTP API server port |
| `-root` | `./dbas-data` | Runtime data directory |
| `-kanban-poll` | `5s` | Kanban daemon poll interval |
| `-feishu-webhook` | `""` | Feishu webhook URL for kanban cards |
| `-max-parallel` | `4` | Max parallel branches |
| `-demo` | `false` | Spawn a demo branch on startup |
| `-version` | `false` | Show version and exit |

---

## Configuration

Copy the example config and edit:

```bash
cp dbas.example.toml dbas.toml
```

The config file specifies LLM providers, executor backend, governance rules, and kanban settings. Cloud providers must use environment variables for credentials — never put API keys in the config file.

### Provider Roles

| Role | Purpose | Example Model |
|------|---------|---------------|
| `decider` | Analyzes tasks, proposes execution plans | gpt-4o |
| `reviewer` | Cross-validates plans against rules | gpt-4o |
| `executor` | Executes approved plans using tools | gpt-4o / deepseek-chat |
| `auditor` | (Optional) Cross-audits evidence | gpt-4o-mini |

---

## Executor Backends

### Native (default)

Uses Reasonix built-in agent with full tool access (bash, file edit, search, web fetch, etc.).

```toml
[executor]
type = "reasonix-native"
```

### Command (external agents)

Bridge to any CLI agent like mini-swe-agent or aider:

```toml
[executor]
type = "command"
command = "mini"
args = ["--model", "openai/deepseek-chat", "--workdir", "{workspace}", "--"]
workdir = "{workspace}"
```

### Placeholders

`args`, `env`, and `workdir` support these placeholders:
`{workspace}`, `{report}`, `{log}`, `{task_file}`, `{branch_id}`, `{task_id}`, `{attempt}`, `{root_hash}`, `{branch_hash}`

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/dbas/state` | All branches and experience slot |
| POST | `/api/dbas/spawn` | Create and execute a new branch |
| POST | `/api/dbas/spawn-parallel` | Create multiple branches in parallel |
| POST | `/api/dbas/reset` | Reset a branch to INIT |
| GET | `/api/dbas/branches/{id}` | Branch detail with report and logs |
| GET | `/api/dbas/slot` | Experience slot configuration |
| POST | `/api/dbas/demo` | Spawn a demo branch |

---

## Verifier Contract

Command executors receive these environment variables:

```text
DBAS_TASK_ID
DBAS_BRANCH_ID
DBAS_ATTEMPT
DBAS_WORKSPACE_DIR
DBAS_REPORT_FILE
DBAS_LOG_FILE
DBAS_TASK_FILE
```

External executors should write all output files under `DBAS_WORKSPACE_DIR` and write a JSON report to `DBAS_REPORT_FILE`:

```json
{
  "task_id": "branch-id",
  "branch_id": "branch-id",
  "status": "success",
  "summary": "what changed",
  "files": ["relative/path"],
  "commands": ["test or build command summary"],
  "test_results": {"go test ./...": "pass"},
  "errors": []
}
```

---

## Dashboard

Open **http://localhost:8080** in your browser.

- **FSM flow** — Real-time state transition visualization
- **Branch cards** — State, model, retries, logs, actions
- **Branch tree** — Parent-child relationships
- **Stats** — Total, active, passed, failed branch counts
- **Spawn panel** — Create branches from the UI

Auto-refreshes every 2 seconds.
