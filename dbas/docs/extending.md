# Extending DBAS — Hermes-Style Extensibility

> DBAS is built for extensibility. Like Hermes, it provides multiple extension mechanisms
> ranging from drop-in Markdown files to full Go plugin interfaces. This guide covers all of them.

---

## Table of Contents

1. [Overview: Five Extension Points](#1-overview-five-extension-points)
2. [Skills (Easiest — No Code)](#2-skills)
3. [Executor Profiles](#3-executor-profiles)
4. [Go Extensions (Plugin Interface)](#4-go-extensions)
5. [MCP Plugins](#5-mcp-plugins)
6. [Custom Provider Backends](#6-custom-provider-backends)
7. [API Reference](#7-api-reference)
8. [Best Practices](#8-best-practices)

---

## 1. Overview: Five Extension Points

| # | Mechanism | Effort | What You Get | Discovery |
|---|-----------|--------|-------------|-----------|
| 1 | **Skills** | 🐣 5 min | Domain-specific Markdown instructions, loaded at runtime | Filesystem |
| 2 | **Executor Profiles** | 🐣 5 min | Pre-built config for aider, mini-swe-agent, opencode | Config file |
| 3 | **Go Extensions** | 🔧 30 min | Register executors, verifiers, notifiers via Go interface | HTTP API |
| 4 | **MCP Plugins** | 🐥 15 min | External tool servers via JSON-RPC over stdio | Config file |
| 5 | **Custom Providers** | 🔧 1 hour | Any OpenAI-compatible LLM backend | Config file |

Pick the mechanism that matches your goal:
- **Want to add domain knowledge?** → Write a Skill
- **Want to use a coding agent as backend?** → Add an Executor Profile
- **Want to add new verification checks?** → Write a Go Extension
- **Want to integrate external tools?** → Add an MCP Plugin
- **Want to use a custom LLM?** → Add a Provider Backend

---

## 2. Skills

Skills are Markdown files with YAML front matter. They inject domain-specific instructions into the agent's context. No Go code needed.

### Quick Start

```markdown
---
name: my-skill
version: 1.0.0
description: What this skill teaches
---

## Instructions

When working on this task, follow these rules:

1. Always do X before Y
2. Never use pattern Z
3. Reference the team's coding standards at docs/standards.md
```

Save as `skills/my-skill.skill.md` and restart DBAS. Skills auto-load at startup.

### Where Skills Load From

By default, DBAS checks these directories:
- `./skills/` (relative to working directory)
- `./dbas/skills/` (relative to reasonix repo root)

### Built-in Skills

DBAS ships with 8 built-in skills:

| Skill | Description |
|-------|-------------|
| `frontend-react` | React/TypeScript frontend development patterns |
| `go-migration` | Database migration best practices |
| `security-audit` | Security audit checklist |
| `rust-security` | Rust unsafe code review |
| `kubernetes-operator` | Kubernetes operator patterns |
| `python-mlops` | MLOps pipeline development |
| `sql-performance` | SQL optimization and schema review |
| `docker-secure` | Docker security hardening |

### Verify Skills Loaded

```bash
curl http://localhost:8080/api/dbas/skills
```

---

## 3. Executor Profiles

Executor profiles configure external coding agents as DBAS execution backends.

### Example: Aider

```toml
[executor]
type = "command"
command = "aider"
args = ["--yes-always", "--message", "-"]
workdir = "{workspace}"
```

### Example: mini-swe-agent

```toml
[executor]
type = "command"
command = "mini"
args = ["--model", "openai/gpt-4o", "--workdir", "{workspace}", "--"]
workdir = "{workspace}"
stdin_format = "text"
```

### Available Placeholders

| Placeholder | Description |
|-------------|-------------|
| `{workspace}` | Branch workspace directory |
| `{report}` | JSON report file path |
| `{log}` | Execution log file path |
| `{task_file}` | Task JSON file path |
| `{branch_id}` | Unique branch identifier |
| `{task_id}` | Task identifier |
| `{attempt}` | Retry attempt number |

Pre-built profiles are available in `dbas/profiles/`.

---

## 4. Go Extensions

Go extensions are the most powerful extension mechanism. They follow the Hermes philosophy: implement an interface, register, done.

### The Extension Interface

```go
type Extension interface {
    Name() string
    Description() string
    Version() string
    RegisterExecutors(r ExecutorRegistry)
    RegisterVerifiers(r VerifierRegistry)
    RegisterNotifiers(r NotifierRegistry)
}
```

### Step 1: Create an Extension

```go
package myextension

import "reasonix/dbas/internal/dbas/plugin"

func init() {
    plugin.Global().RegisterExtension(&MyExtension{})
}

type MyExtension struct{}

func (e *MyExtension) Name() string        { return "my-extension" }
func (e *MyExtension) Description() string  { return "Adds custom verification" }
func (e *MyExtension) Version() string      { return "1.0.0" }

func (e *MyExtension) RegisterExecutors(r plugin.ExecutorRegistry) {
    // Register a custom executor factory
    r.RegisterExecutor("my-executor", func(cfg map[string]interface{}) (interface{}, error) {
        return &MyExecutor{}, nil
    })
}

func (e *MyExtension) RegisterVerifiers(r plugin.VerifierRegistry) {
    r.RegisterVerifier(plugin.VerifierTool{
        Name:    "my-check",
        Command: "my-tool --verify .",
    })
}

func (e *MyExtension) RegisterNotifiers(r plugin.NotifierRegistry) {
    // Register a custom notification channel
}
```

### Step 2: Register It

Add to `runner/runner.go`:

```go
import _ "reasonix/dbas/extensions/myextension"
```

### Step 3: Verify

```bash
# List all registered extensions
curl http://localhost:8080/api/dbas/extensions

# List executors, verifiers, notifiers
curl http://localhost:8080/api/dbas/plugins
```

### Built-in Extensions

DBAS ships with these pre-installed extensions:

| Extension | Provides |
|-----------|----------|
| `native-executors` | reasonix-native, command, docker executors |
| `tool-verifiers` | go-test, pytest, node-test verifiers |
| `channel-notifiers` | console, feishu notifiers |

---

## 5. MCP Plugins

MCP (Model Context Protocol) plugins are external processes that expose tools via JSON-RPC over stdio.

### Configuration

```toml
[[plugins]]
name = "my-mcp-server"
command = "node"
args = ["/path/to/mcp-server/index.js"]
```

### How It Works

1. DBAS starts the external process
2. Communicates via JSON-RPC over the process's stdio
3. Tools exposed by the process become available to executors
4. ```

---

## 6. Custom Provider Backends

DBAS supports any OpenAI-compatible API endpoint. This covers most LLM providers.

### Configuration

```toml
[[providers]]
name = "my-provider"
kind = "openai"
base_url = "https://api.my-provider.com/v1"
model = "my-model"
api_key_env = "MY_API_KEY"
role = "executor"
context_window = 128000
```

### Supported Providers

| Provider | Base URL | Notes |
|----------|----------|-------|
| OpenAI | `https://api.openai.com/v1` | GPT-4o, GPT-4o-mini |
| DeepSeek | `https://api.deepseek.com/v1` | deepseek-chat, deepseek-reasoner |
| Ollama | `http://localhost:11434/v1` | Local models |
| llama.cpp | `http://localhost:8080/v1` | Local GGUF models |
| Any OpenAI-compatible | Your URL | Custom endpoints |

---

## 7. API Reference

### Extension Discovery

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dbas/skills` | List all loaded skills with metadata |
| GET | `/api/dbas/plugins` | List executors, verifiers, notifiers |
| GET | `/api/dbas/extensions` | List all registered extensions with metadata |

### Example Response

```json
GET /api/dbas/extensions

{
  "count": 3,
  "extensions": [
    {
      "name": "native-executors",
      "description": "Built-in executor backends",
      "version": "1.0.0"
    },
    {
      "name": "tool-verifiers",
      "description": "Built-in verification commands",
      "version": "1.0.0"
    },
    {
      "name": "channel-notifiers",
      "description": "Built-in notification channels",
      "version": "1.0.0"
    }
  ]
}
```

---

## 8. Best Practices

### Skill Design
- Keep skills focused on one domain
- Use clear, actionable instructions
- Reference external docs when appropriate
- Version your skills for compatibility

### Extension Design
- Register in `init()` for automatic discovery
- Provide meaningful `Description()` and `Version()`
- Keep extensions stateless where possible
- Use `plugin.Global()` for registration

### Verification Tools
- Match tool commands to your CI pipeline
- Test verifier commands independently before adding them
- Use `{workspace}` placeholder for directory paths

### General
- Prefer Skills over Go Extensions when possible
- Use Executor Profiles to integrate existing tools
- Document custom extensions with skill files
- Share extensions via the community
