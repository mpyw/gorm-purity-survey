# gorm-purity-survey

**Survey documenting GORM's breaking changes in `*gorm.DB` method behavior across patch versions.**

## Purpose

This project exists to **raise awareness** about GORM's unstable API behavior:

- Breaking changes occur in **PATCH versions** (e.g., v1.25.3 → v1.25.4)
- Method semantics (pure/impure, mutable/immutable) change without notice
- No clear documentation of these behavioral changes

This is **NOT** about fixing [gormreuse](https://github.com/mpyw/gormreuse) — that linter already errs on the safe side. This survey serves as **evidence** of GORM's dangerous versioning practices.

## The Problem

GORM's shallow clone behavior has changed multiple times:

```go
// This code's behavior depends on which GORM version you're using:
q := db.Where("active = ?", true)
q.Where("role = ?", "admin").Find(&admins)
q.Where("role = ?", "user").Find(&users)  // Does this include "admin" condition? DEPENDS ON VERSION.
```

The answer varies by patch version. This is unacceptable for a database library.

## Findings

> **TODO**: This section will be populated with survey results.

Expected format:
- List of methods with changed behavior
- Version ranges where each behavior applies
- Specific breaking change examples

## Survey Scope

- **Versions**: v1.20.0 ~ v1.31.1 (latest)
- **Target**: All public methods of `*gorm.DB`

## Survey Criteria

### 1. Pure Classification

A method is **pure** if calling it does not "pollute" the `*gorm.DB` argument or receiver.

**Definition of "pollution" (bomb pattern)**:
- Creating a state that causes problems **when a Finisher is called later**, even if the call itself appears harmless
- Example: `Clauses()` sets up conditions → later `Find()` executes unexpected SQL

```go
// Bomb pattern example
func addClause(db *gorm.DB) {
    db.Clauses(...)  // Nothing happens at this point
}

q := db.Where("x")
addClause(q)        // Bomb planted
q.Find(&results)    // BOOM! Unexpected SQL
```

### 2. Immutable-Return Classification

A method is **immutable-return** if the returned `*gorm.DB` is reusable (safe to branch into multiple code paths).

- `Session()`, `WithContext()`, `Debug()` qualify
- Criteria: whether a new `Statement` is created internally

### 3. Method Introduction Version

Record which version each method was first introduced.

### 4. interface{} Argument Coverage

Methods accepting `interface{}` may have expanded their supported patterns over time. Requires thorough investigation.

## Survey Methodology

### Phase 1: Method Enumeration

List all public methods of `*gorm.DB` from the latest version.

### Phase 2: Source Code Analysis (using Kiri)

Analyze GORM source to understand each method's behavior statically.

### Phase 3: Bisect for Change Detection

For each method, use bisect to identify versions where behavior changed.

```
v1.20 ... v1.25 ... v1.31
    ↓ bisect
Identify change points
```

### Phase 4: Runtime Test Verification

Use Docker to run parallel tests across multiple versions.

## Directory Structure (Planned)

```
gorm-purity-survey/
├── README.md
├── go.mod
├── Dockerfile
├── docker-compose.yml          # Parallel multi-version execution
│
├── methods/                     # Method definitions
│   └── methods.go              # List of methods to survey
│
├── tests/                       # Test code
│   ├── pure_test.go            # Pure classification tests
│   └── immutable_test.go       # Immutable-return classification tests
│
├── scripts/
│   ├── enumerate.go            # Method enumeration script
│   └── bisect.sh               # Bisect automation
│
└── results/                     # Survey results
    ├── v1.20.0.json
    ├── v1.21.0.json
    └── ...
```

## Execution Plan

### Step 1: Environment Setup
- [ ] Create Dockerfile (Go + arbitrary GORM version)
- [ ] Define multiple versions in docker-compose.yml

### Step 2: Method Enumeration
- [ ] List all methods of `*gorm.DB` from latest version
- [ ] Record each method's signature

### Step 3: GORM Source Analysis
- [ ] Analyze method implementations using Kiri
- [ ] Identify introduction version for each method
- [ ] Investigate `interface{}` argument patterns

### Step 4: Test Creation
- [ ] Create tests to classify pure/impure for each method
- [ ] Create tests to classify immutable-return for each method
- [ ] Create bomb pattern detection tests

### Step 5: Cross-Version Bisect
- [ ] Run bisect for each method
- [ ] Record changes per version

### Step 6: Results Summary
- [ ] Create per-version pure/immutable-return compatibility table
- [ ] Formulate policy for reflecting findings in gormreuse

## References

- [gormreuse](https://github.com/mpyw/gormreuse) - GORM *gorm.DB reuse linter
- [GORM Documentation](https://gorm.io/docs/)
- [GORM GitHub](https://github.com/go-gorm/gorm)
