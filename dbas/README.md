[English](#english) | [中文](#chinese)

---

<a name="english"></a>

<h1 align="center">DBAS</h1>
<h3 align="center">The Kubernetes for AI Coding Agents</h3>

[![Build](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/esengine/DeepSeek-Reasonix/actions)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20macOS%20Linux-lightgrey)]()

---

## Why DBAS?

Every AI coding agent — **Claude Code, Cline, Aider, OpenHands, SWE-agent** — is excellent at writing code. But **none of them** answer a fundamental question:

> *"How do I govern, verify, branch, and audit agent work at scale?"*

**DBAS fills this gap.** It doesn't replace coding agents — it orchestrates them. Think of it as **Kubernetes for your AI coding workflow**: a control plane that governs, verifies, and audits every change.

### The Architecture

```
             +-------------+
             |  USER / CI  |
             +------+------+
                    |
         +----------v----------+
         |   DECIDER BRAIN     | <-- Local Qwen / Any LLM
         |   (Proposes plan)   |     Proposes execution plan
         +----------+----------+
                    | proposal JSON
         +----------v----------+
         |  REVIEWER BRAIN     | <-- Local Qwen / Any LLM
         |  (Cross-validates)  |     Validates against rules
         +----------+----------+
                    | decision JSON
         +----------v----------+   +--------------------------+
         |     FSM GATE        |-->|    EXECUTOR BACKEND      |
         |  Hard-enforced      |   |  Reasonix / aider / mini |
         |  state machine      |   |  or any CLI agent        |
         +----------+----------+   +--------------------------+
                    |
         +----------v------------------------------------+
         |          VERIFIER TEAM                        |
         |  . Physical: go test / pytest / eslint        |
         |  . LLM Cross-Audit: independent model         |
         +----------+------------------------------------+
                    | verdict
         +----------v----------+
         |      ARCHIVE        | <-- Tamper-evident audit trail
         +---------------------+
```

## Features

### Dual-Brain Governance
Every execution plan must pass **two independent LLMs**: a Decider proposes, a Reviewer approves. If the Reviewer rejects, the plan is escalated or discarded. No single model can make unilateral decisions.

### FSM-Enforced Lifecycle
State transitions are **hard-enforced at the Go level**. Illegal transitions (e.g., `EXECUTING -> ARCHIVE` skipping `VERIFYING`) are rejected with audit logs. **No LLM prompt can override this.**

### Independent Verification
Two-layer verification:
1. **Physical checks**: `go test`, `pytest`, `npm test`, or custom commands
2. **LLM cross-audit**: An independent model reads the actual output files — not the executor's summary — and validates correctness

### Branch-Based Execution
Every task runs in an isolated branch with its own workspace, log, and artifact tree. Branches form a **radix tree** with hashing chains — inspired by git, optimized for DeepSeek prefix caching.

### Automatic Retry with Escalation
If verification fails, DBAS automatically retries with an upgraded model. Configurable escalation thresholds prevent infinite loops.

### Hermes-Style Extensibility

DBAS provides five levels of extensibility, from drop-in Markdown files to full Go plugin interfaces:

| Level | What You Do | Effort |
|-------|-------------|--------|
| **Skills** | Write a Markdown file with YAML front matter | 5 minutes |
| **Executor Profiles** | Configure an external agent (aider, mini-swe-agent) | 5 minutes |
| **Go Extensions** | Implement the Extension interface | 30 minutes |
| **MCP Plugins** | Connect external tool servers via JSON-RPC | 15 minutes |
| **Custom Providers** | Add any OpenAI-compatible LLM endpoint | Config line |

#### Go Extension Example
```go
type MyExtension struct{}

func (e *MyExtension) Name() string        { return "my-extension" }
func (e *MyExtension) Description() string  { return "Custom verification" }
func (e *MyExtension) Version() string      { return "1.0.0" }

func (e *MyExtension) RegisterExecutors(r plugin.ExecutorRegistry) {
    r.RegisterExecutor("my-executor", myFactory)
}
func (e *MyExtension) RegisterVerifiers(r plugin.VerifierRegistry) {
    r.RegisterVerifier(plugin.VerifierTool{
        Name: "my-check", Command: "my-tool --verify .",
    })
}
func (e *MyExtension) RegisterNotifiers(r plugin.NotifierRegistry) {}

func init() {
    plugin.Global().RegisterExtension(&MyExtension{})
}
```

#### Discovery API
After loading, query everything at runtime:
```bash
curl http://localhost:8080/api/dbas/extensions  # List all extensions with metadata
curl http://localhost:8080/api/dbas/plugins      # Executors, verifiers, notifiers
curl http://localhost:8080/api/dbas/skills       # Loaded skill bundles
```

#### Skill Marketplace (8 Built-in Skills)
`frontend-react` `go-migration` `security-audit` `rust-security`
`kubernetes-operator` `python-mlops` `sql-performance` `docker-secure`

### Real-Time Dashboard
Live kanban-style dashboard showing branch states, FSM flow, logs, and the branch tree. Auto-refreshes every 2 seconds.

## Quickstart

```bash
# No API keys needed -- runs entirely offline
git clone https://github.com/esengine/DeepSeek-Reasonix.git
cd DeepSeek-Reasonix
go build -o dbas.exe ./dbas/cmd/dbas
./dbas.exe --demo --port 8080
```

Then open **http://localhost:8080** — watch 3 branches (happy path, retry logic, FSM enforcement) unfold in real-time.

**No Docker. No API keys. No configuration.**

### With Real LLMs

```bash
cp dbas/dbas.example.toml dbas.toml
export OPENAI_API_KEY="sk-..."
./dbas.exe -config dbas.toml
```

## Extensibility

| Mechanism | Effort | What You Get |
|-----------|--------|-------------|
| Skills | Low | Domain-specific Markdown instruction bundles loaded at runtime |
| Executor Profiles | Low | Pre-built configs for aider, mini-swe-agent, opencode |
| MCP Plugins | Medium | External tool servers via JSON-RPC |
| Custom Extensions | Medium | Register executors, verifiers, notifiers in Go |
| Custom Providers | High | Any OpenAI-compatible endpoint |

## Available Skills

| Skill | Description |
|-------|-------------|
| frontend-react | React/TypeScript frontend development with hooks |
| go-migration | Database migration patterns for Go projects |
| security-audit | Full security audit checklist |
| rust-security | Rust unsafe code review and pointer safety |
| kubernetes-operator | Kubernetes operator controller-runtime patterns |
| python-mlops | MLOps pipelines with experiment tracking |
| sql-performance | SQL query optimization and index analysis |
| docker-secure | Docker image hardening and CVE scanning |

## Comparison

| Feature | DBAS | Claude Code | Aider | OpenHands | Plandex | MetaGPT |
|---------|:----:|:-----------:|:-----:|:---------:|:-------:|:-------:|
| FSM governance | Yes | No | No | No | No | No |
| Dual-brain review | Yes | No | No | No | No | Yes |
| Independent verification | Yes | No | No | No | No | No |
| Branch-based execution | Yes | No | No | No | Yes | No |
| Auto retry + escalation | Yes | No | No | No | No | No |
| Prefix cache optimization | Yes | No | No | No | No | No |
| Offline demo mode | Yes | No | No | No | No | No |
| Kanban dashboard | Yes | No | No | No | No | No |
| Single binary | Yes | No | No | No | Yes | No |
| Hermes-style extensibility | Yes | No | No | No | No | No |
| **Language** | Go | TS | Python | Python | Go | Python |

## CLI Reference

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `dbas.toml` | Path to TOML configuration file |
| `-port` | `8080` | HTTP API server port |
| `-demo` | `false` | Offline demo mode (no API keys needed) |
| `-version` | `false` | Show version and exit |

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/dbas/state` | All branches and experience slot |
| POST | `/api/dbas/spawn` | Create and execute a new branch |
| POST | `/api/dbas/reset` | Reset a branch to INIT |
| GET | `/api/dbas/branches/{id}` | Branch detail with report and logs |
| GET | `/api/dbas/skills` | List all loaded skills with metadata |
| GET | `/api/dbas/plugins` | List executors, verifiers, notifiers |
| GET | `/api/dbas/extensions` | List all extensions with name/desc/version |
| POST | `/api/dbas/demo` | Spawn a demonstration branch |
| GET | `/api/dbas/slot` | Experience slot configuration |

## Roadmap

- **v0.2** — Skill system GA, MCP plugin integration, executor profiles
- **v0.3** — Visual branch tree, sandboxed workspace, Slack integration
- **v0.4** — Production audit mode, evidence export, compliance reporting
- **v1.0** — Stable API, documented extension API, ecosystem CI

## License

Apache 2.0
