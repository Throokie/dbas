# DBAS Architecture

> A technical deep-dive into DBAS — the Disciplined Branching Agent System.

---

## Overview

DBAS is a **governance-first control plane** for AI coding agents. It doesn't write code itself — it orchestrates, verifies, and audits the agents that do.

### Design Principles

1. **Defense in depth**: Three independent gates — Decider, Reviewer, Verifier — prevent any single failure from producing bad output.
2. **Prompt-independent safety**: FSM enforcement is compiled into the Go binary. No prompt injection can bypass it.
3. **Extensibility without fragility**: All extension points are well-defined Go interfaces, not prompt chains.
4. **Observability by default**: Every transition, retry, and verification result is logged and visible in the dashboard.

---

## Core Loop

The system runs a **governance loop** for each task:

```
                    +-----------+
                    |   INPUT   |
                    +-----+-----+
                          |
            +-------------v-------------+
            |         DECIDER            |
            |  (Analyzes task, proposes  |
            |   execution plan as JSON)  |
            +-------------+-------------+
                          | proposal
            +-------------v-------------+
            |         REVIEWER           |
            |  (Validates against rules; |
            |   approves or rejects)     |
            +-------------+-------------+
                          | approved?
                    +-----+-----+
                    |           |
                  YES          NO
                    |           |
         +----------v--+  +----v----+
         |  FSM GATE   |  |ESCALATE |
         |  (enforces  |  |or reject|
         |  lifecycle) |  +----+----+
         +------+------+       |
                |              |
         +------v------+       |
         |  EXECUTOR   |<------+
         |  (agent)    |
         +------+------+
                |
         +------v---------+
         |   VERIFIERS    |
         |  1. Physical   |
         |  2. LLM audit  |
         +------+---------+
                | verdicts
          +-----v-----+
          |  PASS?    |
          +--+-----+--+
             |     |
           YES    NO (retry or fail)
             |
        +----v----+
        | ARCHIVE |
        +---------+
```

---

## Package Architecture

```
dbas/internal/dbas/
├── api/              # HTTP server, middleware, route handlers
│   ├── server.go         - Mux, routes, SPA serving
│   ├── server_ext.go     - Extension/skill/plugin API endpoints
│   ├── server_test.go    - Integration tests
│   └── middleware.go      - Recovery, logging, CORS
│
├── brain/            # AI brain logic
│   ├── decider.go         - Task analysis, plan generation
│   ├── reviewer.go        - Plan validation, cross-check
│   ├── coordinator.go     - Orchestrates decider→reviewer→executor→verifier
│   ├── coordinator_test.go
│   └── prompts.go         - System prompts for each role
│
├── branch/           # Branch lifecycle
│   ├── branch.go          - Branch struct, FSM binding, hashing chain
│   ├── engine.go          - BranchEngine: create, query, manage branches
│   └── engine_test.go
│
├── config/           # Configuration
│   ├── config.go          - Config struct, TOML loading, validation
│   └── config_test.go
│
├── demo/             # Offline demo mode
│   └── demo.go            - 5 demo branches with mock providers
│
├── executor/         # Task execution backends
│   ├── executor.go        - Reasonix-native executor
│   └── executor_test.go
│
├── experienceslot/   # Governance rules & retry logic
│   ├── slot.go            - Experience slot with escalation rules
│   ├── slot_default.go    - Default ruleset
│   └── slot_test.go
│
├── extend/           # Extension system
│   └── extend.go          - 3 built-in extensions with metadata
│
├── feishu/           # Feishu kanban notifier (one-way)
│   └── notifier.go        - Webhook card push
│
├── feishubot/        # Feishu chat bot (bidirectional)
│   ├── bridge.go          - Webhook receiver, command routing, message sending
│   └── bridge_test.go
│
├── fsm/              # Finite state machine
│   ├── fsm.go             - StateMachine with legal transition table
│   └── fsm_test.go
│
├── kanban/           # Kanban dashboard backend
│   ├── daemon.go          - Polling daemon, notifier dispatch
│   └── formatter.go       - Console + Feishu card formatters
│
├── multibranch/      # Parallel branch scheduling
│   ├── scheduler.go       - Sequential + parallel task execution
│   ├── scheduler_test.go
│   ├── radix.go           - Radix tree for branch hierarchy
│   └── radix_test.go
│
├── persistence/      # Storage layer
│   ├── store.go           - Filesystem-based branch store
│   ├── store_test.go
│   └── workspace.go       - Workspace directory management
│
├── plugin/           # Extension registry
│   └── registry.go        - Global registry with lifecycle management
│
├── skill/            # Skill loader
│   └── skill.go           - YAML-front-matter markdown parser + loader
│
└── verifier/         # Verification
    ├── verifier.go         - Physical verification runner
    ├── llm_verifier.go     - LLM-based cross-audit
    ├── llm_verifier_test.go
    └── tools.go            - Pre-built verifier tools (go-test, pytest, etc.)
```

---

## State Machine

The FSM governs branch lifecycle with these states:

```
      +---------+
      |  INIT   |
      +----+----+
           |
      +----v----+
      |  PLAN   |
      +----+----+
           |
      +----v--------+
      |  EXECUTING  |
      +----+--------+
           |
      +----v----------+
      |  VERIFYING    |
      +----+-----+----+
           |     |
         PASS   FAIL
           |     |
      +----v--+  +----v----------+
      | PASS  |  | FAIL_RETREAT  |
      +----+--+  +----+----------+
           |          |
      +----v---+  +---v--------+
      | ARCHIVE|  | ERROR      |
      +--------+  +------------+

Legal transitions (hard-enforced in Go):
  INIT        → PLAN
  PLAN        → EXECUTING
  EXECUTING   → VERIFYING
  VERIFYING   → PASS | FAIL_RETREAT
  FAIL_RETREAT → EXECUTING | ERROR
  PASS        → ARCHIVE
```

**Illegal transitions are rejected at the Go level.** For example, skipping `VERIFYING` to jump directly to `ARCHIVE` is impossible regardless of what any LLM prompt says.

---

## Hashing Chain

Each branch carries a cryptographic hashing chain for integrity:

```
Zone 1 (System-wide):
  RootHash = SHA-256(SystemPrompt + ToolSchemaJSON)

Zone 2 (Per-branch):
  BranchHash = SHA-256(ParentBranchHash + TaskContext)

The hashing chain ensures:
  - Tamper evidence: Any modification to prompts or tool schemas changes RootHash
  - Traceability: Branch hashes form a verifiable tree back to root
  - Prefix cache optimization: Similar branch hashes share DeepSeek cache prefixes
```

---

## Dual-Brain Governance

Every execution plan must pass two independent LLMs:

```
Decider (Role: proponent)
  Input:  Task description + skills + FSM state
  Output: JSON plan with target state, execution plan, tool list
  Prompt: "Analyze the task and propose a governance-compliant execution plan"

Reviewer (Role: validator)
  Input:  Decider's plan + governance rules + experience slot
  Output: JSON decision with approval, modifications, or rejection
  Prompt: "Cross-validate this plan against governance rules and approve or reject"

If Reviewer rejects:
  1. Record the rejection with reason in audit log
  2. If within retry threshold, escalate to stronger model
  3. If beyond threshold, fail the branch
```

---

## Verification

Two-layer verification:

### 1. Physical Verification
Runs actual commands against the workspace:
- `go test ./...` for Go projects
- `pytest` for Python projects
- `npm test` for Node projects
- Custom commands defined in config or extensions

### 2. LLM Cross-Audit
An independent model reads the actual output files — not the executor's summary — and validates:
- Does the output match the task requirements?
- Are there any subtle bugs or regressions?
- Does the output follow best practices from loaded skills?

The cross-audit runs with a different model (or the same model with a different prompt) to avoid confirmation bias.

---

## Extensions

DBAS provides 5 extension mechanisms (see [extending.md](extending.md) for details):

| # | Mechanism | Effort | Runtime Discovery |
|---|-----------|--------|-------------------|
| 1 | Skills (Markdown) | 5 min | Filesystem |
| 2 | Executor Profiles | 5 min | Config file |
| 3 | Go Extensions | 30 min | HTTP API |
| 4 | MCP Plugins | 15 min | Config file |
| 5 | Custom Providers | 1 hour | Config file |

All extensions are discoverable at runtime:
```bash
curl http://localhost:8080/api/dbas/extensions  # List all extensions
curl http://localhost:8080/api/dbas/plugins      # Executors, verifiers, notifiers
curl http://localhost:8080/api/dbas/skills       # Loaded skill bundles
```

---

## Demo Mode

When `--demo` is passed:
- Mock providers replace real LLMs (hardcoded responses that simulate the governance loop)
- 5 branches spawn automatically, each demonstrating a different governance scenario
- The full HTTP API and dashboard are available
- No API keys, no Docker, no configuration required

Demo branches:
| Branch | What It Shows |
|--------|---------------|
| Happy Path | Standard INIT → ARCHIVE lifecycle |
| Retry Path | Two failures followed by successful escalation |
| Error Path | FSM rejection of illegal `EXECUTING → ARCHIVE` |
| Parallel Task | Concurrent subtask execution |
| Skill-Guided | Task guided by the `frontend-react` skill |
