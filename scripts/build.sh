#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="${SCRIPT_DIR}/.."
BIN="${ROOT}/bin"
LIB="${ROOT}/lib"
mkdir -p "$BIN"

echo "Building fit (libfido2 hardware CLI)..."
go build -o "$BIN/fit" ./cmd/fit

echo "Building fit-hello (Windows Hello CLI)..."
go build -o "$BIN/fit-hello" ./cmd/fit-hello || echo "(fit-hello build may be skipped on non-Windows)"

# Copy libraries present (Linux/macOS builds may not need these Windows DLLs)
if compgen -G "$LIB/*.dll" > /dev/null; then
  echo "Copying DLLs..."
  cp "$LIB"/*.dll "$BIN"/
fi

echo "Bin contents:"; ls -l "$BIN"
echo "Done."
