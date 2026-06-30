# DBAS Executor Profiles

Pre-built `[executor]` blocks ready to copy into `dbas.toml`.

## Built-in Profiles

| Profile | Backend | When to Use |
|---------|---------|------------|
| [Native](../BUILD.md) | `reasonix-native` | Default — full tool access, no external deps |
| [mini-swe-agent](./mini-swe-agent.toml) | `command` | Minimal SWE tasks with model control |
| [aider](./aider.toml) | `command` | Codebase-aware pair programming |
| [opencode](./opencode.toml) | `command` | MCP-extensible, CLI-integrated |

## Plugin Profiles

Pre-built `[[plugins]]` blocks for MCP tool servers.

| Plugin | Type | Description |
|--------|------|-------------|
| Filesystem | stdio | File system operations via MCP |
| GitHub | stdio | GitHub API integration |
| PostgreSQL | stdio | Database schema exploration |
| Puppeteer | stdio | Browser automation |

## How to Use

```toml
[executor]
type = "command"
command = "mini"
args = ["--model", "openai/gpt-4o", "--workdir", "{workspace}", "--"]
workdir = "{workspace}"
stdin_format = "text"
```

## Executor Contract

Command executors receive task context through:

- **stdin** — Human-readable instructions or JSON (`stdin_format: "json"`)
- **Environment variables** — `DBAS_WORKSPACE_DIR`, `DBAS_REPORT_FILE`, `DBAS_LOG_FILE`, `DBAS_TASK_FILE`
- **Placeholders** in `args`, `env`, `workdir` — `{workspace}`, `{report}`, `{log}`, `{task_file}`, `{branch_id}`, `{task_id}`, `{attempt}`, `{root_hash}`, `{branch_hash}`

The executor must write output into `{workspace}` and produce a JSON report at `{report}`:

```json
{
  "task_id": "{task_id}",
  "branch_id": "{branch_id}",
  "status": "success|failure|partial",
  "summary": "what changed",
  "files": ["relative/path"],
  "errors": []
}
```

## Writing a Custom Profile

See [../docs/extending.md](../docs/extending.md) for the full extensibility guide.
