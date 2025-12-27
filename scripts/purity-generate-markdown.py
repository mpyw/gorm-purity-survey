#!/usr/bin/env python3
"""Generate Markdown report from purity test JSON results."""

import json
import os
import sys
from pathlib import Path

def main():
    purity_dir = Path(sys.argv[1] if len(sys.argv) > 1 else "purity")

    if not purity_dir.is_dir():
        print(f"Error: {purity_dir} is not a directory", file=sys.stderr)
        sys.exit(1)

    json_files = sorted(purity_dir.glob("v*.json"), key=lambda p: version_key(p.stem))

    if not json_files:
        print(f"Error: No JSON files found in {purity_dir}", file=sys.stderr)
        sys.exit(1)

    # Load all results
    results = {}
    all_methods = set()
    for f in json_files:
        with open(f) as fp:
            data = json.load(fp)
            version = f.stem
            results[version] = data
            all_methods.update(data.get("methods", {}).keys())

    versions = [f.stem for f in json_files]
    all_methods = sorted(all_methods)

    # Print header
    print("# GORM Purity Survey Results")
    print()
    print("This document summarizes the purity behavior of `*gorm.DB` methods across all surveyed GORM versions.")
    print()
    print("## Legend")
    print()
    print("- ✅ **Pure**: Method does NOT pollute the receiver/argument")
    print("- ☠️ **Impure**: Method DOES pollute the receiver/argument")
    print("- ✅ **Immutable-return**: Returned `*gorm.DB` can be safely reused/branched")
    print("- ☠️ **Mutable-return**: Returned `*gorm.DB` is mutable (branches interfere)")
    print()
    print("## Overview")
    print()
    print(f"- **Versions surveyed**: {len(versions)}")
    print(f"- **Version range**: {versions[0]} ~ {versions[-1]}")
    print()

    # Summary table
    print("## Summary by Version")
    print()
    print("| Version | Total | Pure | Impure | Immutable-return |")
    print("|---------|-------|------|--------|------------------|")

    for version in versions:
        data = results[version]
        summary = data.get("summary", {})
        total = summary.get("total_methods", 0)
        pure = summary.get("pure_methods", 0)
        impure = summary.get("impure_methods", 0)
        immutable = summary.get("immutable_count", 0)
        print(f"| {version} | {total} | {pure} | {impure} | {immutable} |")

    print()

    # Method Purity Matrix
    print("## Method Purity Matrix")
    print()
    print("Purity behavior for each method across versions (✅=pure, ☠️=impure, -=N/A):")
    print()

    # Header row
    short_versions = [v.replace("v1.", "") for v in versions]
    print("| Method |", " | ".join(short_versions), "|")
    print("|--------|", "|".join(["------"] * len(versions)), "|")

    # Data rows
    for method in all_methods:
        row = [f"| {method} |"]
        for version in versions:
            methods = results[version].get("methods", {})
            m = methods.get(method, {})
            if not m.get("exists", False):
                row.append(" - |")
            else:
                pure = m.get("pure")
                if pure is True:
                    row.append(" ✅ |")
                elif pure is False:
                    row.append(" ☠️ |")
                else:
                    row.append(" - |")
        print("".join(row))

    print()

    # Immutable-Return Matrix
    print("## Immutable-Return Matrix")
    print()
    print('Whether returned `*gorm.DB` is immutable (✅=immutable, ☠️=mutable, -=N/A):')
    print()

    # Find methods with immutable_return data
    methods_with_immutable = set()
    for version in versions:
        for method, m in results[version].get("methods", {}).items():
            if "immutable_return" in m:
                methods_with_immutable.add(method)
    methods_with_immutable = sorted(methods_with_immutable)

    # Header row
    print("| Method |", " | ".join(short_versions), "|")
    print("|--------|", "|".join(["------"] * len(versions)), "|")

    # Data rows
    for method in methods_with_immutable:
        row = [f"| {method} |"]
        for version in versions:
            methods = results[version].get("methods", {})
            m = methods.get(method, {})
            if not m.get("exists", False):
                row.append(" - |")
            else:
                imm = m.get("immutable_return")
                if imm is True:
                    row.append(" ✅ |")
                elif imm is False:
                    row.append(" ☠️ |")
                else:
                    row.append(" - |")
        print("".join(row))

    print()

    # Purity Changes
    print("## Purity Changes Between Versions")
    print()
    print("Methods whose purity behavior changed between versions:")
    print()

    prev_version = None
    for version in versions:
        if prev_version:
            changes = []
            for method in all_methods:
                prev_data = results[prev_version].get("methods", {}).get(method, {})
                curr_data = results[version].get("methods", {}).get(method, {})
                prev_pure = prev_data.get("pure")
                curr_pure = curr_data.get("pure")

                if prev_pure is not None and curr_pure is not None and prev_pure != curr_pure:
                    if prev_pure is True and curr_pure is False:
                        changes.append(f"- **{method}**: ✅ → ☠️ (became impure)")
                    elif prev_pure is False and curr_pure is True:
                        changes.append(f"- **{method}**: ☠️ → ✅ (became pure)")

            if changes:
                print(f"### {prev_version} → {version}")
                for c in changes:
                    print(c)
                print()
        prev_version = version

    print("---")
    print()
    print("*Generated by gorm-purity-survey*")


def version_key(v):
    """Sort key for version strings like v1.20.0."""
    parts = v.replace("v", "").split(".")
    return tuple(int(p) for p in parts)


if __name__ == "__main__":
    main()
