#!/bin/bash
# Generate Markdown report from purity test JSON results
# Usage: ./scripts/generate-purity-markdown.sh > purity/REPORT.md

set -e

PURITY_DIR="${1:-purity}"

# Check if results exist
if [ ! -d "$PURITY_DIR" ] || [ -z "$(ls -A "$PURITY_DIR"/*.json 2>/dev/null)" ]; then
    echo "Error: No JSON files found in $PURITY_DIR"
    echo "Run ./scripts/purity-all.sh first"
    exit 1
fi

# Get sorted version list
VERSIONS=$(ls "$PURITY_DIR"/*.json 2>/dev/null | xargs -n1 basename | sed 's/.json$//' | sort -V)
VERSION_COUNT=$(echo "$VERSIONS" | wc -l | tr -d ' ')

cat << 'EOF'
# GORM Purity Survey Results

This document summarizes the purity behavior of `*gorm.DB` methods across all surveyed GORM versions.

## Legend

- ✅ **Pure**: Method does NOT pollute the receiver/argument
- ☠️ **Impure**: Method DOES pollute the receiver/argument
- ✅ **Immutable-return**: Returned `*gorm.DB` can be safely reused/branched
- ☠️ **Mutable-return**: Returned `*gorm.DB` is mutable (branches interfere)

## Overview

EOF

echo "- **Versions surveyed**: $VERSION_COUNT"
echo "- **Version range**: $(echo "$VERSIONS" | head -1) ~ $(echo "$VERSIONS" | tail -1)"
echo ""

# Summary table
echo "## Summary by Version"
echo ""
echo "| Version | Total | Pure | Impure | Immutable-return |"
echo "|---------|-------|------|--------|------------------|"

for VERSION in $VERSIONS; do
    FILE="$PURITY_DIR/${VERSION}.json"
    if [ -f "$FILE" ]; then
        TOTAL=$(jq -r '.summary.total_methods // 0' "$FILE")
        PURE=$(jq -r '.summary.pure_methods // 0' "$FILE")
        IMPURE=$(jq -r '.summary.impure_methods // 0' "$FILE")
        IMMUTABLE=$(jq -r '.summary.immutable_count // 0' "$FILE")
        echo "| $VERSION | $TOTAL | $PURE | $IMPURE | $IMMUTABLE |"
    fi
done

echo ""

# Get all methods across all versions
echo "## Method Purity Matrix"
echo ""
echo "Purity behavior for each method across versions (✅=pure, ☠️=impure, -=N/A):"
echo ""

# Get unique method names from all versions
ALL_METHODS=$(for VERSION in $VERSIONS; do
    FILE="$PURITY_DIR/${VERSION}.json"
    if [ -f "$FILE" ]; then
        jq -r '.methods | keys[]' "$FILE" 2>/dev/null
    fi
done | sort -u)

# Create header
echo -n "| Method |"
for VERSION in $VERSIONS; do
    SHORT_VER=$(echo "$VERSION" | sed 's/v1\.//')
    echo -n " $SHORT_VER |"
done
echo ""

# Create separator
echo -n "|--------|"
for VERSION in $VERSIONS; do
    echo -n "------|"
done
echo ""

# Create rows
for METHOD in $ALL_METHODS; do
    echo -n "| $METHOD |"
    for VERSION in $VERSIONS; do
        FILE="$PURITY_DIR/${VERSION}.json"
        if [ -f "$FILE" ]; then
            EXISTS=$(jq -r ".methods.$METHOD.exists // false" "$FILE")
            if [ "$EXISTS" = "true" ]; then
                PURE=$(jq -r ".methods.$METHOD.pure // null" "$FILE")
                if [ "$PURE" = "true" ]; then
                    echo -n " ✅ |"
                elif [ "$PURE" = "false" ]; then
                    echo -n " ☠️ |"
                else
                    echo -n " - |"
                fi
            else
                echo -n " - |"
            fi
        else
            echo -n " - |"
        fi
    done
    echo ""
done

echo ""

# Immutable-return matrix
echo "## Immutable-Return Matrix"
echo ""
echo 'Whether returned `*gorm.DB` is immutable (✅=immutable, ☠️=mutable, -=N/A):'
echo ""

# Create header
echo -n "| Method |"
for VERSION in $VERSIONS; do
    SHORT_VER=$(echo "$VERSION" | sed 's/v1\.//')
    echo -n " $SHORT_VER |"
done
echo ""

# Create separator
echo -n "|--------|"
for VERSION in $VERSIONS; do
    echo -n "------|"
done
echo ""

# Create rows - only for methods that return *gorm.DB
for METHOD in $ALL_METHODS; do
    # Check if any version has immutable_return data for this method
    HAS_IMMUTABLE_DATA=false
    for VERSION in $VERSIONS; do
        FILE="$PURITY_DIR/${VERSION}.json"
        if [ -f "$FILE" ]; then
            IMM=$(jq -r ".methods.$METHOD.immutable_return // null" "$FILE")
            if [ "$IMM" != "null" ]; then
                HAS_IMMUTABLE_DATA=true
                break
            fi
        fi
    done

    if [ "$HAS_IMMUTABLE_DATA" = "true" ]; then
        echo -n "| $METHOD |"
        for VERSION in $VERSIONS; do
            FILE="$PURITY_DIR/${VERSION}.json"
            if [ -f "$FILE" ]; then
                EXISTS=$(jq -r ".methods.$METHOD.exists // false" "$FILE")
                if [ "$EXISTS" = "true" ]; then
                    IMM=$(jq -r ".methods.$METHOD.immutable_return // null" "$FILE")
                    if [ "$IMM" = "true" ]; then
                        echo -n " ✅ |"
                    elif [ "$IMM" = "false" ]; then
                        echo -n " ☠️ |"
                    else
                        echo -n " - |"
                    fi
                else
                    echo -n " - |"
                fi
            else
                echo -n " - |"
            fi
        done
        echo ""
    fi
done

echo ""

# Find changes between versions
echo "## Purity Changes Between Versions"
echo ""
echo "Methods whose purity behavior changed between versions:"
echo ""

PREV_VERSION=""
PREV_FILE=""
for VERSION in $VERSIONS; do
    FILE="$PURITY_DIR/${VERSION}.json"
    if [ -f "$FILE" ] && [ -n "$PREV_FILE" ]; then
        CHANGES=""
        for METHOD in $ALL_METHODS; do
            PREV_PURE=$(jq -r ".methods.$METHOD.pure // null" "$PREV_FILE" 2>/dev/null)
            CURR_PURE=$(jq -r ".methods.$METHOD.pure // null" "$FILE" 2>/dev/null)

            if [ "$PREV_PURE" != "$CURR_PURE" ] && [ "$PREV_PURE" != "null" ] && [ "$CURR_PURE" != "null" ]; then
                if [ "$PREV_PURE" = "true" ] && [ "$CURR_PURE" = "false" ]; then
                    CHANGES="$CHANGES\n- **$METHOD**: ✅ → ☠️ (became impure)"
                elif [ "$PREV_PURE" = "false" ] && [ "$CURR_PURE" = "true" ]; then
                    CHANGES="$CHANGES\n- **$METHOD**: ☠️ → ✅ (became pure)"
                fi
            fi
        done

        if [ -n "$CHANGES" ]; then
            echo "### $PREV_VERSION → $VERSION"
            echo -e "$CHANGES"
            echo ""
        fi
    fi
    PREV_VERSION=$VERSION
    PREV_FILE=$FILE
done

echo "---"
echo ""
echo "*Generated by gorm-purity-survey*"
