#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."
BIN="${ROOT}/bin"
LIB="${ROOT}/lib"
mkdir -p "$BIN"

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-X 'main.buildVersion=${VERSION}'"

# Safe default GOPROXY only if user hasn't set one already.
if [ -z "${GOPROXY:-}" ]; then
  export GOPROXY="https://proxy.golang.org,direct"
  echo "GOPROXY not set; using default: $GOPROXY"
else
  echo "GOPROXY already set: $GOPROXY"
fi

echo "Checking formatting (gofmt)..."
UNFORMATTED=$(gofmt -l . | grep -v '^vendor/' || true)
if [ -n "$UNFORMATTED" ]; then
  echo "Needs formatting:" >&2
  echo "$UNFORMATTED" >&2
fi

echo "Running go mod tidy (non-enforcing)..."
go mod tidy || true

echo "Running go vet..."
if ! go vet ./...; then
  echo "(go vet reported issues; continuing)" >&2
fi

if command -v staticcheck >/dev/null 2>&1; then
  echo "Running staticcheck..."
  if ! staticcheck ./...; then
    echo "(staticcheck reported issues; continuing)" >&2
  fi
else
  echo "staticcheck not installed; skip (install: go install honnef.co/go/tools/cmd/staticcheck@latest)"
fi

echo "Building fit (libfido2 hardware CLI)..."
go build -ldflags "$LDFLAGS" -o "$BIN/fit" ./cmd/fit

echo "Building fit-hello (Windows Hello CLI)..."
go build -ldflags "$LDFLAGS" -o "$BIN/fit-hello" ./cmd/fit-hello || echo "(fit-hello build may be skipped on non-Windows)"

# Copy libraries present (Linux/macOS builds may not need these Windows DLLs)
if compgen -G "$LIB/*.dll" > /dev/null; then
  echo "Copying DLLs..."
  cp "$LIB"/*.dll "$BIN"/
fi

echo "Bin contents:"; ls -l "$BIN"

if command -v govulncheck >/dev/null 2>&1; then
  echo "Running govulncheck..."
  govulncheck ./... || echo "(govulncheck reported issues)"
else
  echo "govulncheck not installed; skip (install: go install golang.org/x/vuln/cmd/govulncheck@latest)"
fi
echo "Done."
