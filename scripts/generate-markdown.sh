#!/bin/bash
# Generate Markdown report from survey JSON results
# Usage: ./scripts/generate-markdown.sh > results/REPORT.md

set -e

RESULTS_DIR="${1:-results}"
OUTPUT_FILE="${2:-results/REPORT.md}"

# Check if results exist
if [ ! -d "$RESULTS_DIR" ] || [ -z "$(ls -A "$RESULTS_DIR"/*.json 2>/dev/null)" ]; then
    echo "Error: No JSON files found in $RESULTS_DIR"
    echo "Run ./scripts/enumerate-all.sh first"
    exit 1
fi

# Get sorted version list
VERSIONS=$(ls "$RESULTS_DIR"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sort -V)
VERSION_COUNT=$(echo "$VERSIONS" | wc -l | tr -d ' ')

cat << 'EOF'
# GORM Purity Survey Results

This document summarizes the method enumeration across all surveyed GORM versions.

## Overview

EOF

echo "- **Versions surveyed**: $VERSION_COUNT"
echo "- **Version range**: $(echo "$VERSIONS" | head -1) ~ $(echo "$VERSIONS" | tail -1)"
echo ""

# Method count comparison
echo "## Method Counts by Version"
echo ""
echo "| Version | \`*gorm.DB\` Methods | Types | Pollution Paths |"
echo "|---------|-------------------|-------|-----------------|"

for VERSION in $VERSIONS; do
    FILE="$RESULTS_DIR/${VERSION}.json"
    if [ -f "$FILE" ]; then
        DB_METHODS=$(jq -r '.types["*gorm.DB"].method_count // 0' "$FILE")
        TYPE_COUNT=$(jq -r '.types | length' "$FILE")
        POLLUTION_COUNT=$(jq -r '.pollution_paths | length' "$FILE")
        echo "| $VERSION | $DB_METHODS | $TYPE_COUNT | $POLLUTION_COUNT |"
    fi
done

echo ""

# Find versions where method count changed
echo "## Method Count Changes"
echo ""
echo "Versions where \`*gorm.DB\` method count changed from previous version:"
echo ""

PREV_COUNT=""
PREV_VERSION=""
for VERSION in $VERSIONS; do
    FILE="$RESULTS_DIR/${VERSION}.json"
    if [ -f "$FILE" ]; then
        COUNT=$(jq -r '.types["*gorm.DB"].method_count // 0' "$FILE")
        if [ -n "$PREV_COUNT" ] && [ "$COUNT" != "$PREV_COUNT" ]; then
            DIFF=$((COUNT - PREV_COUNT))
            if [ $DIFF -gt 0 ]; then
                echo "- **$VERSION**: $PREV_COUNT → $COUNT (+$DIFF methods)"
            else
                echo "- **$VERSION**: $PREV_COUNT → $COUNT ($DIFF methods)"
            fi
        fi
        PREV_COUNT=$COUNT
        PREV_VERSION=$VERSION
    fi
done

echo ""

# New methods per version
echo "## New Methods by Version"
echo ""
echo "Methods added in each version (compared to immediate predecessor):"
echo ""

PREV_FILE=""
for VERSION in $VERSIONS; do
    FILE="$RESULTS_DIR/${VERSION}.json"
    if [ -f "$FILE" ]; then
        if [ -n "$PREV_FILE" ]; then
            # Extract method names and find new ones
            PREV_METHODS=$(jq -r '.types["*gorm.DB"].methods[].name' "$PREV_FILE" 2>/dev/null | sort)
            CURR_METHODS=$(jq -r '.types["*gorm.DB"].methods[].name' "$FILE" 2>/dev/null | sort)
            NEW_METHODS=$(comm -13 <(echo "$PREV_METHODS") <(echo "$CURR_METHODS"))

            if [ -n "$NEW_METHODS" ]; then
                echo "### $VERSION"
                echo ""
                echo "\`\`\`"
                echo "$NEW_METHODS"
                echo "\`\`\`"
                echo ""
            fi
        fi
        PREV_FILE=$FILE
    fi
done

# Generics API section
echo "## Generics API (v1.30+)"
echo ""
echo "Starting from v1.30, GORM introduced Generics API with interfaces that hold internal \`*gorm.DB\`:"
echo ""

# Check if any v1.30+ results exist
for VERSION in $VERSIONS; do
    if [[ "$VERSION" =~ ^v1\.(3[0-9]|[4-9][0-9]) ]]; then
        FILE="$RESULTS_DIR/${VERSION}.json"
        if [ -f "$FILE" ]; then
            echo "### $VERSION"
            echo ""

            # PreloadBuilder
            PB=$(jq -r '.types["gorm.PreloadBuilder"] // empty' "$FILE")
            if [ -n "$PB" ] && [ "$PB" != "null" ]; then
                echo "**PreloadBuilder** methods:"
                echo ""
                jq -r '.types["gorm.PreloadBuilder"].methods[].signature' "$FILE" 2>/dev/null | while read sig; do
                    echo "- \`$sig\`"
                done
                echo ""
            fi

            # JoinBuilder
            JB=$(jq -r '.types["gorm.JoinBuilder"] // empty' "$FILE")
            if [ -n "$JB" ] && [ "$JB" != "null" ]; then
                echo "**JoinBuilder** methods:"
                echo ""
                jq -r '.types["gorm.JoinBuilder"].methods[].signature' "$FILE" 2>/dev/null | while read sig; do
                    echo "- \`$sig\`"
                done
                echo ""
            fi
            break  # Only show one version as example
        fi
    fi
done

# Pollution paths summary
echo "## Pollution Paths Summary"
echo ""
echo "Methods that can potentially pollute \`*gorm.DB\` state:"
echo ""

# Get latest version file
LATEST_FILE="$RESULTS_DIR/$(echo "$VERSIONS" | tail -1).json"
if [ -f "$LATEST_FILE" ]; then
    echo "### Chain Methods (return \`*gorm.DB\`)"
    echo ""
    jq -r '.pollution_paths[] | select(contains("returns *gorm.DB"))' "$LATEST_FILE" 2>/dev/null | head -20 | while read path; do
        echo "- $path"
    done
    echo ""

    echo "### Callback Methods (take \`func(*gorm.DB)\`)"
    echo ""
    jq -r '.pollution_paths[] | select(contains("func with *gorm.DB"))' "$LATEST_FILE" 2>/dev/null | while read path; do
        echo "- $path"
    done
    echo ""
fi

echo "---"
echo ""
echo "*Generated by gorm-purity-survey*"
