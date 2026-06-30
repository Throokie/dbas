# DBAS — Governed Agent Orchestration for AI Coding

<p align="center">
  <strong>Branch, verify, audit, and roll back AI agent work — at any scale.</strong><br>
  A governance-first control plane for AI coding agents, with Hermes-style extensibility.
</p>

<p align="center">
  <a href="#try-it-now">Try It Now (30s)</a> ·
  <a href="#why-dbas">Why DBAS?</a> ·
  <a href="#features">Features</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#extensibility">Extensibility</a> ·
  <a href="#live-demo--screencast">Demo</a> ·
  <a href="#comparison">Comparison</a> ·
  <a href="docs/architecture.md">Docs</a> ·
  <a href="#roadmap">Roadmap</a>
</p>

<p align="center">
  <a href="https://github.com/Throokie/dbas/actions/workflows/dbas-ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/Throokie/dbas/dbas-ci.yml?branch=main&label=CI&logo=github" alt="CI"></a>
  <a href="https://github.com/Throokie/dbas/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-%3E%3D1.25-00ADD8?logo=go" alt="Go Version"></a>
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platform">
  <a href="https://github.com/Throokie/dbas/commits/main"><img src="https://img.shields.io/github/last-commit/Throokie/dbas?logo=github" alt="Last Commit"></a>
  <a href="https://github.com/Throokie/dbas/blob/main/CONTRIBUTING.md"><img src="https://img.shields.io/badge/PRs-welcome-brightgreen" alt="PRs Welcome"></a>
  <a href="https://github.com/Throokie/dbas/stargazers"><img src="https://img.shields.io/github/stars/Throokie/dbas?style=social" alt="GitHub Stars"></a>
</p>

---

## Try It Now (30 seconds, no API keys, no Docker)

```bash
git clone https://github.com/Throokie/dbas.git
cd dbas
go build -o dbas.exe ./dbas/cmd/dbas
./dbas.exe --demo --port 8080
```

Then open **http://localhost:8080** and watch DBAS orchestrate 5 branches through the full governance lifecycle.

> **One binary. No Docker. No API keys. No config. Zero dependencies.**

---

## Live Demo & Screencast

<p align="center">
  <em>Demo recording coming soon. Run <code>./dbas --demo --port 8080</code> now to see it live.</em>
</p>

```text
┌─────────────────────────────────────────────────────────┐
│                  DBAS v0.1.0-alpha                       │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐           │
│  │  Happy   │    │  Retry   │    │  Error   │           │
│  │  Path    │───▶│  Path    │───▶│  Path    │           │
│  │ INIT→ARCH│    │ FAIL→RET │    │ ILLEGAL  │           │
│  └──────────┘    └──────────┘    └──────────┘           │
│  ┌──────────┐    ┌──────────┐                           │
│  │ Parallel │    │  Skill   │                           │
│  │  Tasks   │    │  Guided  │                           │
│  └──────────┘    └──────────┘                           │
│  Dashboard: http://localhost:8080                        │
│  API:       http://localhost:8080/api/dbas/state         │
│  5 branches · 23 skills · 3 extensions                  │
└─────────────────────────────────────────────────────────┘
```

---

## Why DBAS?

Every AI coding agent — **Claude Code, Cline, Aider, Cursor, OpenHands, SWE-agent** — is great at writing code. But none of them answer a fundamental question:

> *"How do I **govern, verify, branch, and audit** agent work when things go wrong?"*

**DBAS fills this gap.** It is not another coding agent. It is a **control plane** that sits above your agents, orchestrating them with hard-enforced governance, independent verification, and tamper-evident audit trails.

Think of it as **Kubernetes for your AI coding workflow**:

| Kubernetes Concept | DBAS Equivalent |
|-------------------|-----------------|
| Container orchestration | Agent orchestration |
| Desired state enforcement | FSM-governed lifecycle |
| Pod health checks | Independent verification |
| RBAC | Dual-brain review (Decider + Reviewer) |
| ConfigMaps / Secrets | Skills (Markdown instruction bundles) |
| CRDs | Plugins, Extensions, Custom Providers |
| kube-apiserver | HTTP API + Dashboard |
| kubectl | Feishu Bot, CLI, API |

---

## Features

| Feature | What It Means |
|---------|---------------|
| **Dual-Brain Governance** | Every plan passes **two independent LLMs**: Decider proposes, Reviewer approves. No single model has unilateral authority. |
| **FSM-Enforced Lifecycle** | State transitions are **hard-coded in Go** — no LLM prompt can skip VERIFYING and jump to ARCHIVE. Illegal transitions are rejected with audit logs. |
| **Independent Verification** | Two-layer verification: physical checks (`go test`, `pytest`, `eslint`) **plus** an independent LLM that reads actual output files — not the executor's summary. |
| **Branch-Based Execution** | Every task runs in an isolated workspace with its own FSM lifecycle, logs, and artifacts. Branches form a **radix tree** with cryptographic hashing chains. |
| **Hermes-Style Extensibility** | **Six mechanisms**: Skills (Markdown), Executor Profiles, Go Extensions, MCP Plugins, Custom Providers, Chat Bot Adapters. All discoverable via HTTP API. |
| **Automatic Retry + Escalation** | On verification failure, DBAS retries with an upgraded model. Configurable escalation thresholds prevent infinite loops. |
| **Real-Time Dashboard** | Live kanban dashboard showing branch states, FSM flow, logs, and branch tree. Auto-refreshes every 2 seconds. Slack/DingTalk/Feishu notifiers. |
| **Zero-Dependency Demo** | `--demo` mode runs entirely offline — no API keys, no Docker, no config. 5 branches unfold in real time. |

---

## Architecture

```text
┌─────────────────────────────────────────────────────┐
│                    DBAS Core                         │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │  Decider  │  │ Reviewer  │  │   Branch Engine   │   │
│  │  (LLM 1)  │  │ (LLM 2)   │  │   FSM · Radix    │   │
│  └─────┬────┘  └────┬─────┘  │   · Hashing Chain │   │
│        └──────┬──────┘        └────────┬─────────┘   │
│               ▼                        ▼              │
│        ┌──────────┐          ┌──────────────────┐    │
│        │ Verifier  │          │   Skill Loader    │    │
│        │ Physical  │          │  (Markdown YAML)  │    │
│        │ + LLM     │          └──────────────────┘    │
│        └──────────┘                                    │
│                                                      │
│  ┌──────────┐  ┌──────────┐                          │
│  │  Kanban   │  │  Feishu  │  ┌──────────────────┐   │
│  │ Dashboard │  │  Bot     │  │   Plugin Registry │   │
│  └──────────┘  └──────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────┘
          │                │                │
          ▼                ▼                ▼
   ┌──────────┐    ┌──────────┐    ┌──────────────────┐
   │  Browser  │    │  Feishu  │    │ External Tools   │
   │ Dashboard │    │  Chat    │    │ (Aider, Claude,  │
   └──────────┘    └──────────┘    │  custom scripts)  │
                                    └──────────────────┘
```

For the full deep-dive: [docs/architecture.md](docs/architecture.md)

---

## Extensibility

DBAS is designed to be extended the way Hermes is — through multiple independent mechanisms that compose together.

### Six Extension Mechanisms

| Mechanism | What It Does | How to Create |
|-----------|-------------|---------------|
| **Skills** | Domain-specific instruction bundles loaded at startup | Write a Markdown file with YAML front matter |
| **Executor Profiles** | Custom execution runtimes | Register via Go API or config TOML |
| **Go Extensions** | Compile-time plugins with lifecycle hooks | Implement the `Extension` interface |
| **MCP Plugins** | Runtime plugin discovery via MCP protocol | Write an MCP-compatible server |
| **Custom Providers** | New LLM backends | Implement the LLM provider interface |
| **Chat Bot Adapters** | Bidirectional chat on any platform | Implement the bot adapter interface |

### Skill Marketplace (23 built-in skills)

| Domain | Skills |
|--------|--------|
| **Frontend** | `frontend-react` · `react-native` |
| **Backend** | `go-migration` · `golang-concurrency` · `api-design` · `graphql-api` |
| **Infrastructure** | `cloud-infrastructure` · `terraform-iac` · `docker-secure` · `kubernetes-operator` |
| **Data & ML** | `data-pipeline` · `database-design` · `machine-learning` · `python-mlops` · `sql-performance` |
| **Security** | `security-audit` · `rust-security` · `code-review` |
| **DevOps** | `devops-cicd` · `observability` |
| **Embedded** | `rust-embedded` |
| **Quality** | `testing-strategy` · `microservices` |

### Built-in Extensions (3)

`sample-executor` · `sample-verifier` · `sample-notifier`

### Extend in 5 Lines

```go
type MyExtension struct{}

func (e *MyExtension) Name() string { return "my-tool" }

func init() {
    plugin.Global().RegisterExtension(&MyExtension{})
}
```

Full guide: [docs/extending.md](docs/extending.md)

---

## Comparison

| Feature | DBAS | Claude Code | Aider | OpenHands | Plandex | MetaGPT | Hermes |
|---------|:----:|:-----------:|:-----:|:---------:|:-------:|:-------:|:------:|
| FSM-governed lifecycle | **Yes** | No | No | No | No | No | No |
| Dual-brain review | **Yes** | No | No | No | No | Yes | No |
| Independent verification | **Yes** | No | No | No | No | No | No |
| Branch-based execution | **Yes** | No | No | No | Yes | No | No |
| Auto retry + escalation | **Yes** | No | No | No | No | No | No |
| Prefix cache optimization | **Yes** | No | No | No | No | No | No |
| Offline demo mode | **Yes** | No | No | No | No | No | No |
| Kanban dashboard | **Yes** | No | No | No | No | No | No |
| Single binary (Go) | **Yes** | No | No | No | Yes | No | No |
| Hermes-style extensibility | **Yes** | No | No | No | No | No | **Yes** |
| Feishu chat bot | **Yes** | No | No | No | No | No | Yes |
| Platform chat bots | In Progress | No | No | No | No | No | Yes |

---

## Roadmap

**v0.1** (current) — Core governance engine, FSM lifecycle, dual-brain review, independent verification, kanban dashboard, 23 skills, 3 extensions, Feishu bot, CI/CD

**v0.2** — Skill API GA, executor profiles (Aider, SWE-agent), MCP plugin integration, visual branch tree, expanded test coverage

**v0.3** — Slack/DingTalk/WeChat bot adapters, sandboxed workspace execution, concurrent executor pool, Grafana dashboards

**v0.4** — Production audit mode with evidence export, compliance reporting (SOC2/SOX), plugin marketplace, performance benchmarks

**v1.0** — Stable API surface, ecosystem CI, documented SDK, community extension registry, real-world case studies

---

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
| GET | `/api/dbas/extensions` | List all extensions with metadata |
| POST | `/api/dbas/demo` | Spawn a demonstration branch |
| GET | `/api/dbas/slot` | Experience slot configuration |
| POST | `/api/feishu/webhook` | Feishu bot event callback |

---

## Development

```bash
# Build
go build -o dbas.exe ./dbas/cmd/dbas

# Test all packages
go test ./dbas/... -v

# Run demo
./dbas.exe --demo --port 8080
```

### Prerequisites

- Go 1.25+
- For real LLM usage: OpenAI-compatible API key

---

## Community & Contributing

- **Issues**: [Bug reports](.github/ISSUE_TEMPLATE/bug_report.md) · [Feature requests](.github/ISSUE_TEMPLATE/feature_request.md) · [Extension requests](.github/ISSUE_TEMPLATE/extension_request.md)
- **PRs**: See [CONTRIBUTING.md](CONTRIBUTING.md)
- **Security**: See [SECURITY.md](SECURITY.md)
- **Extensions**: [docs/extending.md](docs/extending.md) — 6 mechanisms with code samples
- **Discussions**: Start a discussion for questions and ideas

## Documentation

- [Architecture Deep-Dive](docs/architecture.md)
- [Extending DBAS](docs/extending.md)
- [Feishu Integration Guide](docs/feishu-integration.md)
- [Quick Start Guide](START.md)
- [Build Guide](BUILD.md)
- [Deployment Guide](DEPLOY.md)

## License

Apache 2.0 — See [LICENSE](LICENSE)

---

<p align="center">
  <sub>Built with Go 1.25 · Reasonix · and a vision for governed AI</sub>
</p>
