#!/usr/bin/env bash
# MiMoTTS verification script.
# Run from the project root.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== 1. Go tests ==="
cd "$ROOT_DIR/backend"
go test ./... -v

echo ""
echo "=== 2. Go build (linux/arm64, CGO=0) ==="
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags='-s -w' -o /dev/null .

echo ""
echo "=== 3. Go vet ==="
go vet ./...

echo ""
echo "=== 4. Frontend build ==="
cd "$ROOT_DIR/frontend"
if [ ! -d node_modules ]; then
  npm ci
fi
npm run build

echo ""
echo "✅ All checks passed"
