#!/bin/bash
# Enumerate methods for all GORM versions in parallel
# Usage: ./scripts/enumerate-all.sh [parallelism]

set -e

PARALLELISM="${1:-4}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

mkdir -p results

echo "=== GORM Method Enumeration ==="
echo "Versions: $(wc -l < versions.txt | tr -d ' ')"
echo "Parallelism: ${PARALLELISM}"
echo ""

# Make run-version.sh executable
chmod +x scripts/run-version.sh

# Run in parallel
cat versions.txt | xargs -P "$PARALLELISM" -I {} ./scripts/run-version.sh {}

echo ""
echo "=== Complete ==="
echo "Results in: results/"
ls -la results/*.json 2>/dev/null | wc -l | xargs echo "Files generated:"
