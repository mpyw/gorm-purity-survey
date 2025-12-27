# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**gorm-purity-survey** documents GORM's breaking changes in `*gorm.DB` method behavior across patch versions.

### Goal: Raise Awareness

This project is **NOT** about improving gormreuse (which already errs on the safe side).

This is about **documenting and publicizing** how GORM:
- Makes breaking changes in PATCH versions
- Changes method semantics (pure/impure, mutable/immutable) silently
- Provides no documentation of behavioral changes

The survey results will serve as **evidence** of GORM's dangerous versioning practices.

## Key Concepts

### Survey Targets: Two Dimensions

**1. Methods that RETURN `*gorm.DB`** → Check: immutable-return?
**2. Methods that RECEIVE `*gorm.DB`** → Check: pure? (doesn't pollute argument)

Some methods do both (e.g., `Scopes` takes `func(*gorm.DB) *gorm.DB`).

### Pure

A method is **pure** if calling it does NOT pollute the receiver or argument `*gorm.DB`.

**Pollution = Bomb Pattern**: Creating hidden state that explodes when a Finisher is called later.

```go
// NOT pure - plants a bomb
func addClause(db *gorm.DB) {
    db.Clauses(...)  // Looks harmless...
}

q := db.Where("x")
addClause(q)        // Bomb planted
q.Find(&results)    // BOOM! Unexpected SQL
```

### Immutable-Return

A method is **immutable-return** if the returned `*gorm.DB` can be safely reused (branched into multiple code paths).

```go
// Session() is immutable-return
q := db.Session(&gorm.Session{})
q.Where("a").Find(&r1)  // OK
q.Where("b").Find(&r2)  // OK - no interference
```

## Method Categories

### Known Immutable-Return (gormreuse built-in)
- `Session`, `WithContext`, `Debug`, `Begin`

### Survey Targets

1. **Chain Methods** (returns `*gorm.DB`, modifies query state)
   - `Where`, `Or`, `Not`, `Select`, `Order`, `Group`, `Having`, `Joins`, `Preload`, `Clauses`, etc.
   - Question: Do they pollute the receiver? (Almost certainly yes)

2. **Finishers** (executes query)
   - `Find`, `First`, `Take`, `Last`, `Create`, `Update`, `Delete`, `Count`, `Scan`, `Row`, `Rows`, etc.
   - Question: Do they pollute the receiver? (Need investigation)

3. **Methods with `interface{}` args**
   - `Where`, `Or`, `Not`, `Select`, `Order`, `Having`, `Find`, `First`, etc.
   - Question: What patterns are supported per version?

4. **Methods taking `func(*gorm.DB)` callbacks**
   - `Scopes`, `Transaction`, `Connection`, `FindInBatches`, `ToSQL`
   - Question: Is the callback's `*gorm.DB` isolated?

## Docker Build Strategy

### Version Isolation

Each GORM version runs in its own container with isolated `go.mod`:

1. **`.dockerignore`** excludes host `go.mod` / `go.sum`
2. **`go.mod.template`** has `GORM_VERSION_PLACEHOLDER`
3. **`replace` directive** forces exact GORM version (overrides driver requirements)
4. **`go mod tidy`** resolves version-specific indirect dependencies

```dockerfile
# In Dockerfile:
RUN sed "s/GORM_VERSION_PLACEHOLDER/${GORM_VERSION}/g" go.mod.template > go.mod
RUN go mod tidy
```

### Why `replace` is Needed

`gorm.io/driver/mysql` requires minimum GORM versions. Without `replace`:
```
require gorm.io/gorm v1.25.0
# After go mod tidy: gorm v1.25.0 -> v1.25.7 (driver minimum)
```

With `replace`:
```
replace gorm.io/gorm => gorm.io/gorm v1.25.0
# Actual: gorm.io/gorm v1.25.7 => gorm.io/gorm v1.25.0
```

## Development Commands

```bash
# Run enumeration for latest version (host, with Generics API)
go run -tags gorm_v125plus ./scripts/enumerate/... > results/latest.json

# Run enumeration for single version (Docker)
./scripts/run-version.sh v1.25.0

# Run ALL 77 versions in parallel (adjust parallelism as needed)
./scripts/enumerate-all.sh 4

# Quick sample (representative versions only)
docker compose up --build

# Compare methods between two versions
./scripts/diff-versions.sh v1.20.0 v1.30.0

# Generate Markdown report from results
./scripts/generate-markdown.sh > results/REPORT.md
```

## Survey Workflow

### Phase 1: Environment Setup ✓
- [x] Create Dockerfile
- [x] Create compose.yaml with version matrix

### Phase 2: Method Enumeration ✓
- [x] Create enumeration script with **recursive type discovery**
- [x] Generate `results/latest.json`

**Enumerated types** (recursive from `*gorm.DB`):
- `*gorm.DB` (77 methods) - main entry point
- `*gorm.Association` (7 methods) - from `db.Association()`
- `*gorm.Statement` - internal state
- `*gorm.callbacks`, `*gorm.callback`, `*gorm.processor` - callback system
- `*schema.Schema`, `*schema.Field` - schema info
- `clause.Clause`, `clause.Expr`, `clause.Expression` - query builder
- `gorm.Config`, `gorm.Migrator`, `gorm.TableType` - config/migration
- `gorm.PreloadBuilder` (9 methods) - Generics API (v1.30+)
- `gorm.JoinBuilder` (5 methods) - Generics API (v1.30+)

**Generics API version handling**:
- Build tag `gorm_v125plus` controls Generics API type enumeration
- Dockerfile auto-detects GORM version (≥ v1.30) and sets appropriate build tag
- Older versions (< v1.30) use stub that returns empty type list

**Pollution paths detected**:
- Methods that take `*gorm.DB` directly (e.g., `Initialize`, `AfterInitialize`)
- Methods that take `func(*gorm.DB)` (e.g., `Scopes`, `Transaction`, `FindInBatches`)
- Methods that return `*gorm.DB` (chain points - 50+ methods)

### Phase 3: Categorization
- [ ] Categorize all 77 methods by survey priority
- [ ] Identify methods with `interface{}` args for deep investigation
- [ ] Create `methods/categories.go` with method metadata

### Phase 4: Test Creation
For each method category, create tests that detect:
1. **Pollution test**: Does calling the method plant a bomb?
   ```go
   q := db.Session(&gorm.Session{}).Where("base")
   q.SomeMethod(...)  // Call method but discard result
   // Check: does q.Find() include SomeMethod's effect?
   ```

2. **Immutable-return test**: Can the returned value be reused?
   ```go
   q := db.SomeMethod(...)
   q.Where("a").Find(&r1)
   q.Where("b").Find(&r2)
   // Check: are r1 and r2 independent?
   ```

### Phase 5: Version Bisect
For each method where behavior might have changed:
1. Run test on v1.20.0 and latest
2. If results differ, bisect to find change point
3. Record in `results/<version>.json`

### Phase 6: Results Compilation
- [ ] Create per-version compatibility matrix
- [ ] Generate gormreuse configuration recommendations

## Directory Structure

```
gorm-purity-survey/
├── CLAUDE.md              # This file
├── README.md              # Project overview
├── go.mod
├── Dockerfile
├── compose.yaml           # Multi-version test matrix
│
├── scripts/
│   └── enumerate/main.go  # Method enumeration
│
├── methods/               # Method metadata
│   └── categories.go      # Categorized method list
│
├── tests/                 # Test code
│   ├── pollution_test.go  # Pollution detection tests
│   └── immutable_test.go  # Immutable-return tests
│
└── results/               # Survey results per version
    ├── latest.json
    └── ...
```

## Version Matrix

**IMPORTANT**: Breaking changes have occurred in PATCH versions. ALL 76 versions must be surveyed.

See `versions.txt` for the complete list. Key ranges:
- v1.20.x (13 versions)
- v1.21.x (17 versions)
- v1.22.x (6 versions)
- v1.23.x (11 versions)
- v1.24.x (7 versions)
- v1.25.x (13 versions)
- v1.26.x (2 versions)
- v1.30.x (6 versions)
- v1.31.x (2 versions)

Total: 77 versions (v1.20.0 ~ v1.31.1)

## Parallel Execution Strategy

77 versions require efficient parallel testing without freezing the machine.

### Approach: Controlled Parallelism

```bash
# Run N versions in parallel (adjust based on CPU cores)
cat versions.txt | xargs -P 4 -I {} ./scripts/run-version.sh {}
```

### Optimization Techniques

1. **Result Caching**: Skip versions already tested (check `results/<version>.json`)
2. **Docker Build Cache**: Share Go module cache across containers
3. **Incremental Testing**: Only test methods that changed between versions
4. **Signature-First Scan**: Enumerate methods per version first, then only test methods that exist

### Recommended Parallelism

| CPU Cores | Recommended -P |
|-----------|----------------|
| 4         | 2              |
| 8         | 4              |
| 16        | 8              |

### Quick Scan Mode

For initial survey, run enumeration only (no behavior tests):
```bash
./scripts/enumerate-all.sh  # ~5 min for all 77 versions
```

Then run behavior tests only on versions where method signatures differ.

## Testing Strategy

### Why NOT DryRun

DryRun mode may have different code paths than actual execution. We use:
- **sqlmock**: Receives queries without real database
- **Custom Logger**: Captures exact SQL that would be executed

### SQL Capture Logger

```go
// tests/capture/capture.go implements gorm/logger.Interface
cap := capture.New()
db, _ := gorm.Open(mysql.New(...), &gorm.Config{Logger: cap})

// After query execution:
cap.LastSQL()                    // Get last SQL
cap.AllSQL()                     // Get all SQL
cap.ContainsNormalized("admin")  // Check (case-insensitive, whitespace-normalized)
```

### Pollution Test Pattern

"Does calling this method and discarding the result plant a bomb?"

```go
func TestPollution_Where(t *testing.T) {
    db, mock, cap := setupDB(t)

    // 1. Create immutable base
    base := db.Session(&gorm.Session{}).Model(&User{})

    // 2. Call method and DISCARD result (bomb pattern)
    base.Where("role = ?", "admin")  // Result discarded!

    // 3. Execute Finisher on original
    mock.ExpectQuery(".*").WillReturnRows(...)
    base.Find(&users)

    // 4. Check: if "admin" appears, Where polluted the receiver
    if cap.ContainsNormalized("admin") {
        // POLLUTES - not pure
    }
}
```

### Immutable-Return Test Pattern

"Can the returned value be branched without interference?"

```go
func TestImmutableReturn_Where(t *testing.T) {
    db, mock, cap := setupDB(t)

    // 1. Get result from method
    q := db.Model(&User{}).Where("base = ?", true)

    // 2. Branch 1
    q.Where("branch = ?", "one").Find(&r1)

    // 3. Branch 2 - should NOT contain "one"
    cap.Reset()
    q.Where("branch = ?", "two").Find(&r2)

    // 4. Check: if "one" appears, branches interfere
    if cap.ContainsNormalized("one") {
        // NOT immutable-return
    }
}
```

### Callback Isolation Test Pattern

Methods taking `func(*gorm.DB)` need special testing:

```go
func TestCallback_Scopes_Isolation(t *testing.T) {
    // 1. Does callback mutation leak to parent?
    base := db.Session(&gorm.Session{}).Model(&User{})

    scope := func(db *gorm.DB) *gorm.DB {
        return db.Where("in_scope = ?", true)
    }

    base.Scopes(scope).Find(&r1)

    // Query on original - should NOT have "in_scope"
    base.Find(&r2)

    // 2. Is callback *gorm.DB same pointer as base? (dangerous if true)
    var callbackDB *gorm.DB
    scope2 := func(db *gorm.DB) *gorm.DB {
        callbackDB = db
        return db
    }
    base.Scopes(scope2)

    if callbackDB == base {
        // DANGEROUS: same instance
    }
}
```

### Callback Methods Requiring Testing

| Method | Callback Signature | Key Question |
|--------|-------------------|--------------|
| `Scopes` | `...func(*gorm.DB) *gorm.DB` | Is each scope isolated? |
| `Transaction` | `func(*gorm.DB) error` | Does tx mutation leak? |
| `Connection` | `func(*gorm.DB) error` | Does conn mutation leak? |
| `FindInBatches` | `func(*gorm.DB, int) error` | Does batch mutation leak? |
| `ToSQL` | `func(*gorm.DB) *gorm.DB` | Special: no execution |
| `Preload` (func) | `func(*gorm.DB) *gorm.DB` | Only affects Preload query? |

### Generics API - CRITICAL

GORM has a Generics API (`gorm.io/gorm` generics.go) with hidden `*gorm.DB` inside:

**`PreloadBuilder` / `JoinBuilder` (interfaces with internal `*gorm.DB`):**
```go
type preloadBuilder struct {
    db *DB  // HOLDS *gorm.DB internally!
}

func (q *preloadBuilder) Where(...) PreloadBuilder {
    q.db.Where(...)  // DIRECTLY MODIFIES internal db!
    return q
}
```

**Usage in Generics API:**
```go
// ChainInterface[T].Preload passes PreloadBuilder to callback
Preload(association string, query func(db PreloadBuilder) error) ChainInterface[T]

// ChainInterface[T].Joins passes JoinBuilder to callback
Joins(query clause.JoinTarget, on func(db JoinBuilder, ...) error) ChainInterface[T]
```

**Isolation Check:**
```go
// In generics.go, builders are created with NewDB:
q := joinBuilder{db: db.Session(&Session{NewDB: true, ...}).Table(...)}
q := preloadBuilder{db: tx.getInstance()}
```

This *should* isolate from parent, but **version-dependent behavior is suspected**.

**Generic Types to Survey:**
- `Interface[T]`, `ChainInterface[T]`, `CreateInterface[T]`, `ExecInterface[T]`
- `SetUpdateOnlyInterface[T]`, `SetCreateOrUpdateInterface[T]`
- Internal: `g[T]`, `chainG[T]`, `createG[T]`, `execG[T]`, `setCreateOrUpdateG[T]`

All hold `*gorm.DB` internally via `ops []op` pattern.

### Edge Cases to Consider

1. **Finisher side effects**: Does `Find()` reset Statement?
2. **Error state propagation**: Does `AddError()` affect clones?
3. **Method existence per version**: `MapColumns`, `InnerJoins` may not exist in old versions
4. **Statement clone timing**: When exactly is Statement cloned?

## Next Steps

When resuming work on this project:

1. **If Phase 3 incomplete**: Create `methods/categories.go` with priority categorization
2. **If Phase 4 incomplete**: Create pollution/immutable tests for high-priority methods
3. **If Phase 5 incomplete**: Run bisect on methods with suspected version differences
4. **If Phase 6 incomplete**: Compile results into actionable gormreuse recommendations

## References

- [gormreuse](https://github.com/mpyw/gormreuse) - Target linter
- [GORM Docs](https://gorm.io/docs/)
- [GORM GitHub](https://github.com/go-gorm/gorm)
- [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock)
