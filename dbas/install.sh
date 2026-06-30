#!/bin/bash
# DBAS Installer
set -e
echo "==> DBAS Installer"
if ! command -v go &>/dev/null; then
  echo "Error: Go required. Install from https://go.dev/dl/"
  exit 1
fi
TMP=\`mktemp -d\`
trap "rm -rf \$TMP" EXIT
git clone --depth=1 https://github.com/esengine/DeepSeek-Reasonix.git \$TMP
cd \$TMP
go build -ldflags="-s -w" -o /usr/local/bin/dbas ./dbas/cmd/dbas
echo "Installed to /usr/local/bin/dbas"
echo ""
echo "Run: dbas --demo --port 8080"
