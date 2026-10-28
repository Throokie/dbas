# DBAS — Governed Agent Orchestration for AI Coding

<p align="center">
  <strong>Branch, verify, audit, and roll back AI agent work — at any scale.</strong>
</p>

<p align="center">
  <!-- Restore the workflow badge after dbas-ci.yml has a successful run on main. -->
  <img src="https://img.shields.io/badge/CI-not%20yet%20green-lightgrey" alt="CI pending">
  <a href="https://github.com/Throokie/dbas/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/go-%3E%3D1.25-00ADD8?logo=go" alt="Go Version"></a>
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platform">
  <a href="https://github.com/Throokie/dbas/stargazers"><img src="https://img.shields.io/github/stars/Throokie/dbas?style=social" alt="GitHub Stars"></a>
</p>

---

**DBAS** (Disciplined Branching Agent System) is a governance-first control plane for AI coding agents. It orchestrates, verifies, and audits agent work with an FSM-enforced lifecycle, independent verification, and Hermes-style extensibility.

```bash
git clone https://github.com/Throokie/dbas.git
cd dbas
go build -o dbas.exe ./dbas/cmd/dbas
./dbas.exe --demo --port 8080
# Open http://localhost:8080
```

> **One binary. No Docker. No API keys. 30 seconds.**

---

## Key Differentiators

- **FSM-governed lifecycle** — State transitions are hard-coded in Go, not prompt-dependent
- **Dual-brain review** — Every plan passes two independent LLMs (Decider + Reviewer)
- **Independent verification** — Physical checks + LLM cross-audit of actual output files
- **6 extension mechanisms** — Skills, Executors, Extensions, MCP Plugins, Custom Providers, Chat Bots
- **23 built-in skills** — Frontend, Backend, Infrastructure, Data/ML, Security, DevOps, Embedded
- **Zero-dependency demo** — Runs entirely offline with no API keys

---

## Quick Start

| Step | Command |
|------|---------|
| **Clone** | `git clone https://github.com/Throokie/dbas.git` |
| **Build** | `go build -o dbas.exe ./dbas/cmd/dbas` |
| **Run demo** | `./dbas.exe --demo --port 8080` |
| **Dashboard** | Open http://localhost:8080 |
| **Run with LLMs** | `cp dbas/dbas.example.toml dbas.toml` then `./dbas.exe -config dbas.toml` |

---

## Documentation

| Document | Content |
|----------|---------|
| [Full README](dbas/README.md) | Complete documentation with features, architecture, comparison, roadmap |
| [Architecture](dbas/docs/architecture.md) | Deep-dive into FSM, dual-brain, hashing chain, package map |
| [Extending DBAS](dbas/docs/extending.md) | 6 extension mechanisms with code samples |
| [Feishu Integration](dbas/docs/feishu-integration.md) | Kanban notifications + bidirectional chat bot setup |
| [Quick Start Guide](dbas/START.md) | Detailed getting started walkthrough |
| [Deployment Guide](dbas/DEPLOY.md) | Production deployment options |

---

## License

Apache 2.0
