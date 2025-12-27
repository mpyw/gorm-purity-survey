#!/bin/bash
# Enumerate methods for all GORM versions in parallel
# Usage: ./scripts/enumerate-all.sh [parallelism]
#
# Each version has built-in retry logic (see run-version.sh)

PARALLELISM="${1:-4}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

mkdir -p results

TOTAL_VERSIONS=$(wc -l < versions.txt | tr -d ' ')

echo "=== GORM Method Enumeration ==="
echo "Versions: ${TOTAL_VERSIONS}"
echo "Parallelism: ${PARALLELISM}"
echo ""

# Make run-version.sh executable
chmod +x scripts/run-version.sh

# Run in parallel (don't use set -e, collect failures)
cat versions.txt | xargs -P "$PARALLELISM" -I {} ./scripts/run-version.sh {} || true

echo ""
echo "=== Summary ==="

COMPLETED=$(ls results/*.json 2>/dev/null | wc -l | tr -d ' ')
echo "Completed: ${COMPLETED}/${TOTAL_VERSIONS}"

if [ "$COMPLETED" -lt "$TOTAL_VERSIONS" ]; then
    echo ""
    echo "Failed versions:"
    comm -23 <(cat versions.txt | sort) <(ls results/*.json 2>/dev/null | xargs -n1 basename 2>/dev/null | sed 's/.json//' | sort)
    exit 1
else
    echo "All versions completed successfully!"
fi
