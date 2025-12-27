# gorm-purity-survey

**Survey documenting GORM's breaking changes in `*gorm.DB` method behavior across patch versions.**

## Quick Links

- **[USAGE_GUIDE.md](./USAGE_GUIDE.md)** - Practical guide: "Which version breaks with what pattern?"
- **[purity/REPORT.md](./purity/REPORT.md)** - Full purity survey results
- **[methods/REPORT.md](./methods/REPORT.md)** - Method enumeration by version

## Purpose

This project **documents and publicizes** GORM's unstable API behavior:

- Breaking changes occur in **PATCH versions** (e.g., v1.25.11 → v1.25.12)
- Method semantics (pure/impure, mutable/immutable) change without notice
- No clear documentation of these behavioral changes

This survey serves as **evidence** of GORM's dangerous versioning practices.

## The Problem

GORM's method behavior has changed multiple times across patch versions:

```go
// This code's behavior depends on which GORM version you're using:
q := db.Where("active = ?", true)
q.Where("role = ?", "admin").Find(&admins)
q.Where("role = ?", "user").Find(&users)  // Does this include "admin"? DEPENDS ON VERSION.
```

## Key Findings

### Version-Specific Breaking Changes

| Version | Change | Impact |
|---------|--------|--------|
| **v1.20.7** | Session clone: 1→2 | Session() now fully isolates |
| **v1.20.8** | Preload callback clone: 1→2 | Preload callbacks now isolate |
| **v1.21.8** | Delete/Update became pure | Safe to discard return values |
| **v1.23.2** | Begin clone: 2→1 | Begin() less isolated than Session() |
| **v1.25.7** | Limit/Offset became pure | Safe to discard return values |
| **v1.30.0** | **Preload callback clone: 2→0** | **REGRESSION: #7662** |

### Purity Classification

Based on testing all 77 versions (v1.20.0 ~ v1.31.1):

| Category | Symbol | Count (v1.31.1) |
|----------|--------|-----------------|
| Pure | ✅ | 26 methods |
| Impure-Accumulate | ☠️ | ~15 methods |
| Impure-Overwrite | ⚠️ | ~6 methods |

**Impure-Accumulate (dangerous)**: `Where`, `Or`, `Not`, `Order`, `Joins`
**Impure-Overwrite (less dangerous)**: `Select`, `Distinct`

### The Golden Rule

**Always call `Session(&gorm.Session{})` before branching a `*gorm.DB`.**

```go
// SAFE in all versions
base := db.Session(&gorm.Session{})
base.Where("x").Find(&r1)  // OK
base.Where("y").Find(&r2)  // OK - no interference
```

See [USAGE_GUIDE.md](./USAGE_GUIDE.md) for detailed patterns.

## Survey Scope

- **Versions**: 77 versions (v1.20.0 ~ v1.31.1)
- **Target**: All public methods of `*gorm.DB`
- **Tests**: Pure classification, immutable-return, callback isolation

## Survey Results

### Method Enumeration (methods/)

| Version | `*gorm.DB` Methods |
|---------|-------------------|
| v1.20.0 | 70 |
| v1.21.0 | 73 |
| v1.25.11 | 76 |
| v1.31.1 | 77 |

New methods added: `CreateInBatches` (v1.20.7), `ToSQL` (v1.22.3), `Connection` (v1.22.5), `InnerJoins` (v1.24.3), `MapColumns` (v1.25.11)

### Purity Test Results (purity/)

Full results in [purity/REPORT.md](./purity/REPORT.md), including:
- Per-method pure/impure status by version
- Clone values (0/1/2) for return values and callbacks
- Impure mode (accumulate vs overwrite)
- Version-specific regressions

## Directory Structure

```
gorm-purity-survey/
├── README.md              # This file
├── USAGE_GUIDE.md         # Practical usage patterns guide
├── CLAUDE.md              # Development instructions
│
├── methods/               # Method enumeration results
│   ├── REPORT.md         # Summary report
│   └── v*.json           # Per-version method lists
│
├── purity/                # Purity test results
│   ├── REPORT.md         # Full survey report
│   └── v*.json           # Per-version purity data
│
├── scripts/
│   ├── methods/          # Enumeration code
│   ├── purity/           # Purity test code
│   ├── *-run.sh          # Single version scripts
│   ├── *-all.sh          # Parallel all-version scripts
│   └── *-generate-*.py   # Report generators
│
├── Dockerfile.methods     # Method enumeration container
├── Dockerfile.purity      # Purity testing container
└── versions.txt           # All 77 GORM versions
```

## Running the Survey

```bash
# Method enumeration (all 77 versions, 4 parallel)
./scripts/methods-all.sh 4
./scripts/methods-generate-markdown.sh > methods/REPORT.md

# Purity testing (all 77 versions, 4 parallel)
./scripts/purity-all.sh 4
python3 scripts/purity-generate-markdown.py > purity/REPORT.md

# Single version
./scripts/methods-run.sh v1.31.1
./scripts/purity-run.sh v1.31.1
```

## References

- [gormreuse](https://github.com/mpyw/gormreuse) - GORM `*gorm.DB` reuse linter
- [GORM Documentation](https://gorm.io/docs/)
- [GORM GitHub](https://github.com/go-gorm/gorm)

### Key GORM Issues

- [#7662](https://github.com/go-gorm/gorm/issues/7662) - Preload callback clone=0 (v1.30.0+)
- [#7594](https://github.com/go-gorm/gorm/issues/7594) - InnerJoins+Preload duplicate JOIN
- [#7027](https://github.com/go-gorm/gorm/pull/7027) - AfterQuery Joins clearing fix
