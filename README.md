# DBAS — Standalone Repository

This is the standalone extraction of DBAS from the Reasonix monorepo.

## Quick Start

```bash
go build -o dbas.exe ./dbas/cmd/dbas
./dbas.exe --demo --port 8080
```

## Building

```bash
go build ./dbas/...
go test ./dbas/...
```

## Structure

- `dbas/` — DBAS source code
- `internal/` — Reasonix dependencies (vendored for standalone use)
