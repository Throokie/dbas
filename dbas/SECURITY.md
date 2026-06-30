# DBAS Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes (dev) |

## Reporting a Vulnerability

DBAS is a governance control plane for AI coding agents. If you discover a security vulnerability, please do NOT file a public issue.

Instead, report via:
1. **GitHub Security Advisories**: Use the "Report a Vulnerability" tab on the repository
2. **Email**: Contact the maintainers through the GitHub repository

We aim to respond within 48 hours and publish a fix within 7 days.

## Security Considerations for AI Agent Governance

DBAS sits in a critical position as a governance layer. Key security principles:

### API Key Management
- Never store API keys in configuration files or code
- Use environment variables referenced by `api_key_env` in TOML config
- Rotate keys regularly

### Audit Trail
- All branch state transitions are logged with timestamps
- Execution logs are persisted for forensic analysis
- Verification failures are recorded for compliance review

### FSM Integrity
- State transitions are hard-enforced at the Go runtime level
- Illegal transitions (e.g., skipping VERIFYING) are rejected with audit logs
- No LLM prompt or external input can override the FSM

### Workspace Isolation
- Each branch runs in its own isolated workspace directory
- Branch workspaces are not shared between parallel executions
- Workspace files are cleaned up based on retention policy

### Supply Chain
- All dependencies are vendored and reviewed
- Docker images use distroless base images
- Binary releases are checksum-verified
