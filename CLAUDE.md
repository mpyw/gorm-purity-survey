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
# === Method Enumeration ===
# Run for single version
./scripts/methods-run.sh v1.25.0

# Run ALL 77 versions in parallel
./scripts/methods-all.sh 4

# Generate report
./scripts/methods-generate-markdown.sh > methods/REPORT.md

# === Purity Testing ===
# Run for single version
./scripts/purity-run.sh v1.25.0

# Run ALL 77 versions in parallel
./scripts/purity-all.sh 4

# Generate report
./scripts/purity-generate-markdown.sh > purity/REPORT.md
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
├── Dockerfile.methods     # Method enumeration Docker
├── Dockerfile.purity      # Purity testing Docker
│
├── scripts/
│   ├── methods/           # Method enumeration Go code
│   │   └── main.go
│   ├── methods-run.sh     # Run enumeration for single version
│   ├── methods-all.sh     # Run enumeration for all versions
│   ├── methods-generate-markdown.sh
│   │
│   ├── purity/            # Purity testing Go code
│   │   ├── main.go
│   │   └── version_*.go   # Version-specific tests (build tags)
│   ├── purity-run.sh      # Run purity test for single version
│   ├── purity-all.sh      # Run purity test for all versions
│   └── purity-generate-markdown.sh
│
├── methods/               # Method enumeration results (JSON per version)
│
├── purity/                # Purity test results (JSON per version)
│
└── tests/                 # Test utilities
    └── capture/           # SQL capture logger
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

### Pure Test Pattern

"Does calling this method pollute the receiver?"

**IMPORTANT**: Test from MUTABLE base (no Session), not immutable.

```go
func TestPure_Where(t *testing.T) {
    db, mock, cap := setupDB(t)

    // 1. Create MUTABLE base (NO Session!)
    q := db.Model(&User{}).Where("base")

    // 2. Call method and DISCARD result
    q.Where("marker")  // Does this pollute q?

    // 3. Execute Finisher on original
    mock.ExpectQuery(".*").WillReturnRows(...)
    q.Find(&users)

    // 4. Check: if "marker" appears, method polluted receiver
    if cap.ContainsNormalized("marker") {
        // NOT pure - pollutes receiver
    }
}
```

### Immutable-Return Test Pattern

"Can the returned value be branched without interference?"

```go
func TestImmutableReturn_Where(t *testing.T) {
    db, mock, cap := setupDB(t)

    // 1. Get the return value to test
    q := db.Model(&User{}).Where("base")

    // 2. Branch 1 - execute first
    mock.ExpectQuery(".*").WillReturnRows(...)
    q.Where("branch_one").Find(&r1)

    // 3. Branch 2 - should NOT contain "branch_one"
    cap.Reset()
    mock.ExpectQuery(".*").WillReturnRows(...)
    q.Where("branch_two").Find(&r2)

    // 4. Check: if "branch_one" appears, return value is mutable
    if cap.ContainsNormalized("branch_one") {
        // NOT immutable-return - branches interfere
    }
}
```

### Callback Argument Immutability Test Pattern

"Can the callback's `*gorm.DB` argument be branched without interference?"

For methods like Preload that take `func(*gorm.DB) *gorm.DB` callbacks:

```go
func TestCallbackImmutable_Preload(t *testing.T) {
    db, mock, cap := setupDB(t)

    // Use recover() since callback support varies by version
    defer func() {
        if r := recover(); r != nil {
            // Callback not supported in this version
        }
    }()

    var callCount int
    callback := func(tx *gorm.DB) *gorm.DB {
        callCount++
        // Branch from callback's *gorm.DB
        if callCount == 1 {
            return tx.Where("callback_marker")
        }
        return tx
    }

    q := db.Model(&User{}).Preload("Association", callback)

    // First execution
    mock.ExpectQuery(".*").WillReturnRows(...)
    q.Find(&r1)

    // Second execution - check if callback's db was polluted
    cap.Reset()
    mock.ExpectQuery(".*").WillReturnRows(...)
    q.Find(&r2)

    // If "callback_marker" appears twice, callback db is mutable
    if cap.CountNormalized("callback_marker") > 1 {
        // Callback argument is MUTABLE (v1.30.0+ regression!)
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

## Current Status

### Phase 3 Complete: Basic Purity Tests ✓

Basic purity tests implemented and run on all 77 versions:
- **Pure test**: Mutable base, discard result, check pollution
- **Immutable-return test**: Branch interference check
- **Callback-arg immutable test**: Preload callback accumulation

**Key Findings from Phase 3:**
- v1.21.0 briefly made all methods pure (reverted in v1.21.1)
- v1.21.8: Delete/Update/Updates became pure
- v1.25.7: Limit/Offset became pure
- Latest (v1.31.1): 26 pure, 21 impure methods

### Phase 4: Extended Tests (TODO)

Additional test dimensions needed to catch known regressions:

#### 4.1 Finisher Reuse Test (Joins Preservation)
**Issue**: PR #7027 fixed Count() clearing Joins, but behavior varies by version

```go
// Test: After Finisher, are Joins preserved for next query?
q := db.Model(&User{}).Joins("Profile")
q.Count(&count)    // 1st finisher
q.Find(&users)     // 2nd finisher - Joins still there?

// Check SQL of 2nd query for "Profile" join
// Possible outcomes:
//   - Joins preserved (expected)
//   - Joins cleared (PR #7027 pre-fix bug)
//   - Joins duplicated (regression)
```

**Versions to watch**: v1.25.x where PR #7027 was applied

#### 4.2 InnerJoins + Preload Duplicate Test
**Issue**: PR #7014 (v1.25.12) broke InnerJoins + nested Preload

```go
// Test: InnerJoins + Preload on nested relations
db.Model(&Comic{}).
   InnerJoins("Book.MstBook").
   Preload("Book.MstBook.Episodes").
   Find(&results)

// Check SQL for duplicate table names
// PostgreSQL error: "table name X specified more than once"
// Possible outcomes:
//   - Single JOIN per table (correct)
//   - Duplicate JOINs (PR #7014 regression)
```

**Versions to watch**: v1.25.12+, v1.30.x, v1.31.x

#### 4.3 Preload Callback Argument Mutation
**Issue**: GitHub #7662 - Preload callback's `*gorm.DB` became mutable in v1.30.0

```go
// Test: Preload callback arg accumulates state across executions
callback := func(tx *gorm.DB) *gorm.DB {
    return tx.Where("marker_col = ?", true)
}
q := db.Model(&User{}).Preload("Profile", callback)

q.Find(&r1)  // 1st execution: marker appears once
q.Find(&r2)  // 2nd execution: marker appears once or twice?

// If marker appears twice in 2nd execution SQL:
//   Callback arg is MUTABLE (regression in v1.30.0+)
```

**Versions to watch**: v1.30.0+ (clone=0 issue)

### Known GORM Issues Timeline

| Version | PR/Issue | Problem | Status |
|---------|----------|---------|--------|
| v1.21.0 | - | All methods briefly pure | Reverted in v1.21.1 |
| v1.25.8 | PR #6877 | Fixed nested Preload+Join panic | Removed reflect.Pointer support |
| v1.25.9 | PR #6990 | Merged nested preload queries | Performance optimization |
| v1.25.11 | PR #6957 | Re-added reflect.Pointer | Fixed "unsupported data" error |
| v1.25.12 | PR #7014 | Use reflect.Append + nil skip | **Broke InnerJoins+Preload** |
| v1.25.12 | PR #7027 | Fix AfterQuery Joins clearing | Fixed Count clearing Joins |
| v1.30.0+ | #7662 | Preload callback clone=0 | **Callback arg mutable** |
| v1.31.1 | - | Latest version | PR #7014 issue NOT fixed |

### Implementation Plan

#### Step 1: Add Test Models
Need models with nested relations for InnerJoins+Preload test:
```go
type Comic struct {
    ID     uint
    BookID uint
    Book   Book
}
type Book struct {
    ID      uint
    MstBook MstBook
}
type MstBook struct {
    ID       uint
    Episodes []Episode
}
```

#### Step 2: Implement New Tests
Add to `scripts/purity/main.go`:
- `testFinisherReuse_Joins()` - Count→Find with Joins
- `testInnerJoinsPreloadDuplicate()` - Nested InnerJoins+Preload
- Enhance `testPreload()` callback test

#### Step 3: Update JSON Schema
Add new result fields:
```json
{
  "methods": {
    "Joins": {
      "finisher_preserves": true/false,
      "innerjoins_preload_safe": true/false
    },
    "Preload": {
      "callback_arg_immutable": true/false
    }
  }
}
```

#### Step 4: Re-run All Versions
```bash
rm purity/*.json
./scripts/purity-all.sh 4
python3 scripts/purity-generate-markdown.py > purity/REPORT.md
```

#### Step 5: Update Report Generator
Add new sections:
- Finisher Reuse Matrix
- InnerJoins+Preload Safety Matrix
- Version-specific regression warnings

### Alternative Approach: godump for Direct State Inspection

**Problem with current SQL-based detection:**
- Indirect: relies on SQL output to infer internal state changes
- May miss changes to fields that don't affect SQL
- Complex marker string matching

**Better approach using godump (https://github.com/goforj/godump):**

```go
import "github.com/goforj/godump"

// Direct state comparison
before := godump.Sprint(db.Statement)
db.Where("x")  // discard result
after := godump.Sprint(db.Statement)

if before != after {
    // Method polluted receiver - can see exactly what changed
}
```

**Implementation Strategy:**

#### Step 0: Identify Relevant Fields (Do First!)

Run godump on one version (e.g., v1.31.1) to identify which `Statement` fields matter:

```go
// Test script to identify relevant fields
db := setupDB()
base := db.Model(&User{})

fmt.Println("=== Before Where ===")
godump.Dump(base.Statement)

base.Where("x")

fmt.Println("=== After Where (discarded) ===")
godump.Dump(base.Statement)
```

**Expected relevant fields in `*gorm.Statement`:**
- `Clauses` - WHERE, ORDER BY, etc.
- `Joins` - JOIN information
- `Preloads` - Preload configurations
- `Selects` / `Omits` - Column selection
- `Table` / `Model` - Table info
- `Dest` - Destination pointer

**Fields to IGNORE (likely noise):**
- `DB` - Pointer to parent (circular)
- `Context` - Context object
- `ConnPool` - Connection pool
- `Schema` - Cached schema (may change on first access)
- `Settings` - sync.Map (hard to compare)

#### Step 1: Create Field Extractor

```go
type StatementSnapshot struct {
    Clauses   map[string]interface{}
    Joins     []string  // join names
    Preloads  map[string]bool
    Table     string
    Selects   []string
    Omits     []string
}

func snapshotStatement(stmt *gorm.Statement) StatementSnapshot {
    // Extract only relevant fields
    snapshot := StatementSnapshot{...}
    return snapshot
}

func (s StatementSnapshot) Equals(other StatementSnapshot) bool {
    return reflect.DeepEqual(s, other)
}
```

#### Step 2: Rewrite Tests with State Comparison

```go
func testPure_Where() {
    db := setupDB()
    base := db.Model(&User{})

    before := snapshotStatement(base.Statement)
    base.Where("x")  // discard
    after := snapshotStatement(base.Statement)

    pure := before.Equals(after)
    // No need to execute query or check SQL!
}
```

**Advantages:**
- Direct detection of any state change
- No need for SQL marker strings
- Can detect changes that don't affect SQL output
- Clearer test logic

**Considerations:**
- Need to handle version-specific Statement struct changes
- Some fields may be legitimately lazy-initialized
- Circular references need careful handling

### When Resuming This Work

1. Read this section first
2. **Run godump exploration first** to identify relevant Statement fields
3. Create `snapshotStatement()` function
4. Rewrite tests with state comparison approach
5. Key versions to verify: v1.25.11, v1.25.12, v1.30.0, v1.31.1

## References

### Tools
- [gormreuse](https://github.com/mpyw/gormreuse) - Target linter
- [godump](https://github.com/goforj/godump) - Struct inspection for state comparison
- [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) - SQL mocking

### GORM Documentation
- [GORM Docs](https://gorm.io/docs/)
- [GORM GitHub](https://github.com/go-gorm/gorm)

### Key GORM Issues (Breaking Changes)
- [#7662](https://github.com/go-gorm/gorm/issues/7662) - Preload callback clone=0 regression (v1.30.0+)
- [#7594](https://github.com/go-gorm/gorm/issues/7594) - InnerJoins+Preload duplicate JOIN (v1.25.12+)
- [#7027](https://github.com/go-gorm/gorm/pull/7027) - AfterQuery Joins clearing fix
- [#7014](https://github.com/go-gorm/gorm/pull/7014) - reflect.Append change (broke InnerJoins+Preload)
- [#6957](https://github.com/go-gorm/gorm/pull/6957) - reflect.Pointer re-added
- [#6877](https://github.com/go-gorm/gorm/pull/6877) - Nested Preload+Join panic fix
