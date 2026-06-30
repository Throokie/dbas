#!/usr/bin/env bash
# DBAS Quick Start — try DBAS in 30 seconds
# No API keys, no Docker, no configuration needed.
set -euo pipefail

echo ""
echo "  DBAS Quick Start"
echo "  ================"
echo ""

# Check Go
if ! command -v go &>/dev/null; then
    echo "  ERROR: Go is not installed."
    echo "  Install Go 1.25+: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
echo "  ✓ Go $GO_VERSION detected"

# Navigate to repo root (parent of dbas/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

echo "  ✓ Building DBAS..."

go build -o dbas ./dbas/cmd/dbas 2>/dev/null || {
    echo "  ERROR: Build failed. Make sure go.mod exists in $REPO_ROOT"
    exit 1
}

echo "  ✓ Build complete"
echo ""
echo "  Starting DBAS in DEMO mode..."
echo "  Dashboard: http://localhost:8080"
echo "  API:       http://localhost:8080/api/dbas/state"
echo ""

./dbas --demo --port 8080
