# Contributing to DBAS

We love contributions! DBAS aims to be the governance layer for AI coding agents, and we need the community to make it great.

## Quick Start for Contributors

### Prerequisites
- Go 1.25+
- Git
- An OpenAI-compatible API key (for non-demo mode)

### Local Development

```bash
git clone https://github.com/esengine/DeepSeek-Reasonix.git
cd DeepSeek-Reasonix
go build -o dbas.exe ./dbas/cmd/dbas
./dbas.exe --demo --port 8080
```

### Project Structure

```
dbas/
  cmd/dbas/           # Entry point (main.go)
  runner/             # Shared startup logic
  internal/dbas/
    api/              # HTTP API server + middleware
    brain/            # Decider, Reviewer, Coordinator
    branch/           # Branch engine + workspace
    config/           # TOML configuration loading
    demo/             # Offline demo mode
    executor/         # Reasonix-native executor
    experienceslot/   # Governance rules
    extend/           # Built-in extensions
    feishu/           # Feishu notifier
    fsm/              # State machine
    kanban/           # Real-time dashboard daemon
    multibranch/      # Parallel branch scheduler
    persistence/      # Branch store + workspace
    plugin/           # Extension registry
    skill/            # Markdown skill loader
    verifier/         # Verification tools
  profiles/           # Executor backend configs
  skills/             # Domain skill bundles (Markdown)
  frontend/dist/      # Dashboard HTML/CSS/JS
```

## Ways to Contribute

### 1. Add a Skill (Easiest)
Create a `.skill.md` file in `dbas/skills/` with YAML front matter:

```markdown
---
name: my-skill
version: 1.0.0
description: What this skill teaches
---

## Instructions
...
```

Skills load automatically at startup. No code changes needed.

### 2. Add an Executor Profile
Create a `.toml` file in `dbas/profiles/`:

```toml
[executor]
type = "command"
command = "my-agent"
args = ["--workdir", "{workspace}"]
workdir = "{workspace}"
```

### 3. Write Tests
We need more test coverage! Check for untested packages:
```bash
go test ./dbas/... -count=1 -timeout 120s
```

### 4. Improve the Frontend
The dashboard lives in `dbas/frontend/dist/index.html` — a single-file HTML app with embedded CSS/JS.

### 5. Add Documentation
Good documentation is as valuable as code. Check the `dbas/docs/` directory.

### 6. Report a Bug
Open an issue with:
- DBAS version and platform
- Steps to reproduce
- Expected vs actual behavior
- Full logs from `dbas-data/branches/*/execution.log`

## Development Workflow

1. Fork the repo
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make your changes
4. Run tests: `go test ./dbas/... -count=1 -timeout 120s`
5. Run vet: `go vet ./dbas/...`
6. Run demo: `go run ./dbas/cmd/dbas --demo --port 8080`
7. Commit with a descriptive message
8. Push and open a PR

## Code Style

- Follow standard Go formatting (`gofmt`)
- Use expressive variable names
- Add comments for non-obvious logic
- Keep functions focused and small
- Test error paths, not just happy paths

## PR Guidelines

- One feature/fix per PR
- Include tests for new functionality
- Update documentation if behavior changes
- Keep the diff focused — no unrelated refactoring
- Reference the issue number in the description

## Community

- Be respectful and inclusive
- Help review others' PRs
- Answer questions in issues
- Share your skills and profiles

## License

By contributing, you agree that your contributions will be licensed under Apache 2.0.
