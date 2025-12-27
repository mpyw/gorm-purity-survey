#!/bin/bash
# Compare method signatures between two versions
# Usage: ./scripts/diff-versions.sh v1.20.0 v1.31.1

set -e

V1="${1:?First version required}"
V2="${2:?Second version required}"

METHODS_DIR="methods"

if [ ! -f "${METHODS_DIR}/${V1}.json" ]; then
    echo "Error: ${METHODS_DIR}/${V1}.json not found"
    exit 1
fi

if [ ! -f "${METHODS_DIR}/${V2}.json" ]; then
    echo "Error: ${METHODS_DIR}/${V2}.json not found"
    exit 1
fi

echo "=== Method Diff: ${V1} vs ${V2} ==="
echo ""

# Extract signatures and compare (from *gorm.DB type)
jq -r '.types["*gorm.DB"].methods[].signature' "${METHODS_DIR}/${V1}.json" | sort > /tmp/v1_sigs.txt
jq -r '.types["*gorm.DB"].methods[].signature' "${METHODS_DIR}/${V2}.json" | sort > /tmp/v2_sigs.txt

echo "--- Only in ${V1} ---"
comm -23 /tmp/v1_sigs.txt /tmp/v2_sigs.txt

echo ""
echo "--- Only in ${V2} ---"
comm -13 /tmp/v1_sigs.txt /tmp/v2_sigs.txt

echo ""
echo "--- Common methods ---"
comm -12 /tmp/v1_sigs.txt /tmp/v2_sigs.txt | wc -l | xargs echo "Count:"
