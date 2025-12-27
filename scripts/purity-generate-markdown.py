#!/usr/bin/env python3
"""Generate Markdown report from purity test JSON results."""

import json
import sys
from pathlib import Path


def version_key(v):
    """Sort key for version strings like v1.20.0."""
    parts = v.replace("v", "").split(".")
    return tuple(int(p) for p in parts)


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
    short_versions = [v.replace("v1.", "") for v in versions]

    # Print header
    print("# GORM Purity Survey Results")
    print()
    print("This document summarizes the purity behavior of `*gorm.DB` methods across all surveyed GORM versions.")
    print()
    print("## Legend")
    print()
    print("### Purity")
    print("- ✅ **Pure**: Method does NOT pollute the receiver")
    print("- ⚠️ **Impure-Overwrite**: Pollutes receiver, but repeated calls overwrite (less dangerous)")
    print("- ☠️ **Impure-Accumulate**: Pollutes receiver, repeated calls stack up (DANGEROUS)")
    print()
    print("### Immutable-Return")
    print("- ✅ **Immutable**: Returned `*gorm.DB` can be safely reused/branched")
    print("- ☠️ **Mutable**: Returned `*gorm.DB` is mutable (branches interfere)")
    print()
    print("### Clone Values")
    print("- `0`: No cloning (DANGEROUS - mutations leak)")
    print("- `1`: Clone Statement with empty Clauses")
    print("- `2`: Full clone (Statement.clone(), keeps Clauses)")
    print("- `-1`: Not detected / N/A")
    print()
    print("## Overview")
    print()
    print(f"- **Versions surveyed**: {len(versions)}")
    print(f"- **Version range**: {versions[0]} ~ {versions[-1]}")
    print()

    # Summary table
    print("## Summary by Version")
    print()
    print("| Version | Total | Pure | Impure | Immutable |")
    print("|---------|-------|------|--------|-----------|")

    for version in versions:
        data = results[version]
        summary = data.get("summary", {})
        total = summary.get("total_methods", 0)
        pure = summary.get("pure_methods", 0)
        impure = summary.get("impure_methods", 0)
        immutable = summary.get("immutable_count", 0)
        print(f"| {version} | {total} | {pure} | {impure} | {immutable} |")

    print()

    # Method Purity Matrix (with impure_mode)
    print("## Method Purity Matrix")
    print()
    print("Purity behavior for each method across versions:")
    print("- ✅ = pure")
    print("- ⚠️ = impure-overwrite")
    print("- ☠️ = impure-accumulate")
    print("- ❓ = impure (mode unknown)")
    print("- `-` = N/A")
    print()

    # Header row
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
                impure_mode = m.get("impure_mode")
                if pure is True:
                    row.append(" ✅ |")
                elif pure is False:
                    if impure_mode == "accumulate":
                        row.append(" ☠️ |")
                    elif impure_mode == "overwrite":
                        row.append(" ⚠️ |")
                    else:
                        row.append(" ❓ |")
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

    # Clone Value Matrix (return_clone)
    print("## Return Clone Value Matrix")
    print()
    print("The `clone` field value of returned `*gorm.DB` (determines immutability):")
    print()

    # Find methods with return_clone data
    methods_with_return_clone = set()
    for version in versions:
        for method, m in results[version].get("methods", {}).items():
            if "return_clone" in m:
                methods_with_return_clone.add(method)
    methods_with_return_clone = sorted(methods_with_return_clone)

    if methods_with_return_clone:
        # Header row
        print("| Method |", " | ".join(short_versions), "|")
        print("|--------|", "|".join(["------"] * len(versions)), "|")

        # Data rows
        for method in methods_with_return_clone:
            row = [f"| {method} |"]
            for version in versions:
                methods = results[version].get("methods", {})
                m = methods.get(method, {})
                if not m.get("exists", False):
                    row.append(" - |")
                else:
                    clone = m.get("return_clone")
                    if clone is None:
                        row.append(" - |")
                    elif clone == 0:
                        row.append(" **0** |")  # Dangerous
                    elif clone == 1:
                        row.append(" 1 |")
                    elif clone == 2:
                        row.append(" 2 |")
                    else:
                        row.append(f" {clone} |")
            print("".join(row))

        print()

    # Callback Clone Value Matrix
    print("## Callback Clone Value Matrix")
    print()
    print("The `clone` field value of `*gorm.DB` passed to callbacks:")
    print()

    # Find methods with callback_clone data
    methods_with_callback_clone = set()
    for version in versions:
        for method, m in results[version].get("methods", {}).items():
            if "callback_clone" in m:
                methods_with_callback_clone.add(method)
    methods_with_callback_clone = sorted(methods_with_callback_clone)

    if methods_with_callback_clone:
        # Header row
        print("| Method |", " | ".join(short_versions), "|")
        print("|--------|", "|".join(["------"] * len(versions)), "|")

        # Data rows
        for method in methods_with_callback_clone:
            row = [f"| {method} |"]
            for version in versions:
                methods = results[version].get("methods", {})
                m = methods.get(method, {})
                if not m.get("exists", False):
                    row.append(" - |")
                else:
                    clone = m.get("callback_clone")
                    if clone is None:
                        row.append(" - |")
                    elif clone == -1:
                        row.append(" -1 |")  # Not detected
                    elif clone == 0:
                        row.append(" **0** |")  # Dangerous
                    elif clone == 1:
                        row.append(" 1 |")
                    elif clone == 2:
                        row.append(" 2 |")
                    else:
                        row.append(f" {clone} |")
            print("".join(row))

        print()

    # Callback Argument Immutability Matrix
    print("## Callback Argument Immutability Matrix")
    print()
    print("Whether callback's `*gorm.DB` argument is immutable (✅=immutable, ☠️=mutable):")
    print()

    # Find methods with callback_arg_immutable data
    methods_with_callback_immutable = set()
    for version in versions:
        for method, m in results[version].get("methods", {}).items():
            if "callback_arg_immutable" in m:
                methods_with_callback_immutable.add(method)
    methods_with_callback_immutable = sorted(methods_with_callback_immutable)

    if methods_with_callback_immutable:
        # Header row
        print("| Method |", " | ".join(short_versions), "|")
        print("|--------|", "|".join(["------"] * len(versions)), "|")

        # Data rows
        for method in methods_with_callback_immutable:
            row = [f"| {method} |"]
            for version in versions:
                methods = results[version].get("methods", {})
                m = methods.get(method, {})
                if not m.get("exists", False):
                    row.append(" - |")
                else:
                    imm = m.get("callback_arg_immutable")
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

    # Clone Value Changes
    print("## Clone Value Changes Between Versions")
    print()
    print("Methods whose clone values changed between versions:")
    print()

    prev_version = None
    for version in versions:
        if prev_version:
            changes = []
            for method in all_methods:
                prev_data = results[prev_version].get("methods", {}).get(method, {})
                curr_data = results[version].get("methods", {}).get(method, {})

                # Check return_clone
                prev_clone = prev_data.get("return_clone")
                curr_clone = curr_data.get("return_clone")
                if prev_clone is not None and curr_clone is not None and prev_clone != curr_clone:
                    changes.append(f"- **{method}** return_clone: {prev_clone} → {curr_clone}")

                # Check callback_clone
                prev_cb = prev_data.get("callback_clone")
                curr_cb = curr_data.get("callback_clone")
                if prev_cb is not None and curr_cb is not None and prev_cb != curr_cb:
                    changes.append(f"- **{method}** callback_clone: {prev_cb} → {curr_cb}")

            if changes:
                print(f"### {prev_version} → {version}")
                for c in changes:
                    print(c)
                print()
        prev_version = version

    # Key Findings Summary
    print("## Key Findings Summary")
    print()
    print("### Session/Begin Clone Value Swap")
    print()
    print("| Version Range | Session | Begin |")
    print("|---------------|---------|-------|")

    # Detect session/begin clone value ranges
    session_begin_ranges = []
    current_session = None
    current_begin = None
    range_start = None

    for version in versions:
        m = results[version].get("methods", {})
        session_clone = m.get("Session", {}).get("return_clone")
        begin_clone = m.get("Begin", {}).get("return_clone")

        if session_clone != current_session or begin_clone != current_begin:
            if range_start is not None:
                session_begin_ranges.append((range_start, prev_version, current_session, current_begin))
            range_start = version
            current_session = session_clone
            current_begin = begin_clone
        prev_version = version

    if range_start is not None:
        session_begin_ranges.append((range_start, versions[-1], current_session, current_begin))

    for start, end, session, begin in session_begin_ranges:
        if start == end:
            print(f"| {start} | {session} | {begin} |")
        else:
            print(f"| {start} ~ {end} | {session} | {begin} |")

    print()

    # Scopes Clone Value Changes
    print("### Scopes Clone Value Changes")
    print()
    print("| Version Range | return_clone | callback_clone |")
    print("|---------------|--------------|----------------|")

    scopes_ranges = []
    current_return = None
    current_cb = None
    range_start = None

    for version in versions:
        m = results[version].get("methods", {})
        return_clone = m.get("Scopes", {}).get("return_clone")
        cb_clone = m.get("Scopes", {}).get("callback_clone")

        if return_clone != current_return or cb_clone != current_cb:
            if range_start is not None:
                scopes_ranges.append((range_start, prev_version, current_return, current_cb))
            range_start = version
            current_return = return_clone
            current_cb = cb_clone
        prev_version = version

    if range_start is not None:
        scopes_ranges.append((range_start, versions[-1], current_return, current_cb))

    for start, end, ret, cb in scopes_ranges:
        cb_display = f"**{cb}**" if cb == 0 else str(cb) if cb is not None else "-"
        if start == end:
            print(f"| {start} | {ret} | {cb_display} |")
        else:
            print(f"| {start} ~ {end} | {ret} | {cb_display} |")

    print()

    print("---")
    print()
    print("*Generated by gorm-purity-survey*")


if __name__ == "__main__":
    main()
