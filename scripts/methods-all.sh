#!/bin/bash
# Enumerate methods for all GORM versions in parallel
# Usage: ./scripts/methods-all.sh [parallelism]
#
# Each version has built-in retry logic (see methods-run.sh)

PARALLELISM="${1:-4}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

mkdir -p methods

TOTAL_VERSIONS=$(wc -l < versions.txt | tr -d ' ')

echo "=== GORM Method Enumeration ==="
echo "Versions: ${TOTAL_VERSIONS}"
echo "Parallelism: ${PARALLELISM}"
echo ""

# Make methods-run.sh executable
chmod +x scripts/methods-run.sh

# Run in parallel (don't use set -e, collect failures)
cat versions.txt | xargs -P "$PARALLELISM" -I {} ./scripts/methods-run.sh {} || true

echo ""
echo "=== Summary ==="

COMPLETED=$(ls methods/*.json 2>/dev/null | wc -l | tr -d ' ')
echo "Completed: ${COMPLETED}/${TOTAL_VERSIONS}"

if [ "$COMPLETED" -lt "$TOTAL_VERSIONS" ]; then
    echo ""
    echo "Failed versions:"
    comm -23 <(cat versions.txt | sort) <(ls methods/*.json 2>/dev/null | xargs -n1 basename 2>/dev/null | sed 's/.json//' | sort)
    exit 1
else
    echo "All versions completed successfully!"
fi
