# GORM Usage Guide: Safe Patterns by Version

This guide explains which GORM versions break with which usage patterns. Based on purity testing of all 77 versions (v1.20.0 ~ v1.31.1).

## TL;DR: The Golden Rule

**Always call `Session(&gorm.Session{})` before branching a `*gorm.DB`.**

```go
// SAFE in all versions
base := db.Session(&gorm.Session{})
branch1 := base.Where("x").Find(&r1)  // OK
branch2 := base.Where("y").Find(&r2)  // OK - no interference

// DANGEROUS in all versions
base := db.Model(&User{})
branch1 := base.Where("x")
branch2 := base.Where("y")
branch1.Find(&r1)  // May contain "y" condition!
```

---

## Quick Reference: What Breaks in Each Version

| Version Range | Breaking Issue | Pattern That Breaks |
|---------------|----------------|---------------------|
| **All versions** | Chain methods pollute receiver | `db.Where()` discarding return value |
| **v1.20.0 ~ v1.20.6** | Session clone=1 (partial) | Session() + nested Joins may leak |
| **v1.20.0 ~ v1.20.7** | Preload callback clone=1 | Preload callbacks may accumulate |
| **v1.20.0 ~ v1.21.7** | Delete/Update impure | `db.Delete()` pollutes receiver |
| **v1.20.0 ~ v1.25.6** | Limit/Offset impure | `db.Limit()` pollutes receiver |
| **v1.23.2+** | Begin clone=1 (not 2) | Transaction Begin() shares some state |
| **v1.30.0+** | Preload callback clone=0 | **#7662** Preload callbacks accumulate across reuse |

---

## Detailed Patterns by Category

### 1. Chain Method Pollution (All Versions)

**Problem**: Chain methods like `Where`, `Order`, `Joins` modify the receiver, not just the return value.

```go
// DANGEROUS - works in NO version
func addFilter(db *gorm.DB) {
    db.Where("status = ?", "active")  // Return value discarded = bomb planted
}

q := db.Model(&User{})
addFilter(q)
q.Find(&users)  // SQL includes "status = active" - surprise!

// SAFE - works in ALL versions
func addFilter(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "active")  // Return value used
}

q := db.Model(&User{})
q = addFilter(q)  // Must reassign
q.Find(&users)
```

**Impure Methods (pollute receiver in all versions)**:
- ☠️ `Where`, `Or`, `Not` - accumulate conditions
- ☠️ `Order` - accumulates ordering
- ☠️ `Joins`, `InnerJoins` - accumulates joins
- ⚠️ `Select` - overwrites (less dangerous)
- ⚠️ `Distinct` - overwrites

### 2. Branching Without Session (All Versions)

**Problem**: Without `Session()`, branching creates interfering queries.

```go
// DANGEROUS - breaks in ALL versions
q := db.Model(&User{}).Where("base")
q.Where("branch_a").Find(&r1)
q.Where("branch_b").Find(&r2)  // Contains BOTH branch_a AND branch_b!

// SAFE - works in ALL versions
q := db.Session(&gorm.Session{}).Model(&User{}).Where("base")
q.Where("branch_a").Find(&r1)
q.Where("branch_b").Find(&r2)  // Contains only base + branch_b
```

### 3. Preload Callback Mutation (v1.30.0+ Regression)

**GitHub Issue**: [#7662](https://github.com/go-gorm/gorm/issues/7662)

**Problem**: In v1.30.0+, Preload callback's `*gorm.DB` is NOT cloned (clone=0), causing conditions to accumulate.

```go
// DANGEROUS in v1.30.0+ (clone=0)
// SAFE in v1.20.8 ~ v1.26.1 (clone=2)
callback := func(tx *gorm.DB) *gorm.DB {
    return tx.Where("visible = ?", true)
}

q := db.Model(&User{}).Preload("Posts", callback)
q.Find(&users1)  // Posts WHERE visible = true (OK)
q.Find(&users2)  // Posts WHERE visible = true AND visible = true (BUG!)
```

**Version Timeline**:
| Version Range | Preload callback_clone | Behavior |
|---------------|------------------------|----------|
| v1.20.0 ~ v1.20.7 | 1 | Partial clone |
| v1.20.8 ~ v1.26.1 | 2 | **Full clone (SAFE)** |
| v1.30.0 ~ v1.31.1 | 0 | **No clone (BROKEN)** |

**Workaround for v1.30.0+**:
```go
// Create fresh query each time
for i := 0; i < 2; i++ {
    q := db.Session(&gorm.Session{}).Model(&User{}).Preload("Posts", callback)
    q.Find(&users)
}
```

### 4. Session vs Begin Clone Difference

**Problem**: Session() and Begin() use different clone strategies in some versions.

| Version Range | Session clone | Begin clone | Notes |
|---------------|---------------|-------------|-------|
| v1.20.0 ~ v1.20.6 | 1 | 2 | Session partial, Begin full |
| v1.20.7 ~ v1.23.1 | 2 | 2 | Both full (best) |
| v1.23.2 ~ v1.31.1 | 2 | 1 | Session full, Begin partial |

**Recommendation**: Always use `Session(&gorm.Session{})` for branching, not `Begin()`.

### 5. Limit/Offset Became Pure in v1.25.7

```go
// DANGEROUS in v1.20.0 ~ v1.25.6
q := db.Model(&User{})
q.Limit(10)  // Pollutes q
q.Find(&users)  // Has LIMIT 10 even though Limit() return was discarded

// SAFE in v1.25.7+
q := db.Model(&User{})
q.Limit(10)  // Does NOT pollute q (pure)
q.Find(&users)  // No LIMIT
```

### 6. Delete/Update/Updates Became Pure in v1.21.8

```go
// DANGEROUS in v1.20.0 ~ v1.21.7
q := db.Model(&User{}).Where("x")
q.Delete(&User{})  // Pollutes q
q.Find(&users)  // Affected by Delete's internal state

// SAFE in v1.21.8+
q := db.Model(&User{}).Where("x")
q.Delete(&User{})  // Does NOT pollute q (pure)
q.Find(&users)  // Clean query
```

---

## Safe Patterns That Work in ALL Versions

### Pattern 1: Always Use Session() at Branch Points

```go
func getBaseQuery(db *gorm.DB) *gorm.DB {
    return db.Session(&gorm.Session{}).Model(&User{}).Where("deleted_at IS NULL")
}

// Each branch is independent
activeUsers := getBaseQuery(db).Where("status = ?", "active").Find(&active)
inactiveUsers := getBaseQuery(db).Where("status = ?", "inactive").Find(&inactive)
```

### Pattern 2: Never Discard Chain Method Return Values

```go
// BAD
func applyFilters(db *gorm.DB) {
    db.Where("x")  // Discarded!
    db.Order("y")  // Discarded!
}

// GOOD
func applyFilters(db *gorm.DB) *gorm.DB {
    return db.Where("x").Order("y")
}
```

### Pattern 3: Create Fresh Queries for Loops

```go
// BAD - accumulates conditions
q := db.Model(&User{})
for _, filter := range filters {
    q.Where(filter)  // Accumulates!
}

// GOOD - fresh query each iteration
for _, filter := range filters {
    q := db.Session(&gorm.Session{}).Model(&User{}).Where(filter)
    q.Find(&results)
}
```

### Pattern 4: Use Scopes Carefully

```go
// Scopes pass *gorm.DB to callback - callback mutations may leak
// in some versions

// SAFE: Use Session inside scope
func ActiveScope(db *gorm.DB) *gorm.DB {
    return db.Where("active = ?", true)  // OK - returns new value
}

db.Scopes(ActiveScope).Find(&users)
```

---

## Version-Specific Recommendations

### v1.20.x Users
- Upgrade to at least v1.20.8 for Preload callback clone=2
- Be aware Session() only has clone=1 until v1.20.7
- Delete/Update pollute receiver

### v1.21.x ~ v1.25.6 Users
- Delete/Update safe from v1.21.8
- Limit/Offset still pollute - always use return values
- Preload callbacks are safe (clone=2)

### v1.25.7 ~ v1.26.x Users
- **Best versions for purity** - most methods are pure
- Limit/Offset became pure
- Preload callbacks still safe (clone=2)

### v1.30.x ~ v1.31.x Users
- **CRITICAL**: Preload callback clone=0 regression (#7662)
- Do not reuse queries with Preload callbacks
- Always create fresh queries for each Find()

---

## Method Quick Reference

| Method | Pure? | Mode | Safe to Discard Return? |
|--------|-------|------|------------------------|
| `Where` | ☠️ No | accumulate | NO - pollutes receiver |
| `Or` | ☠️ No | accumulate | NO |
| `Not` | ☠️ No | accumulate | NO |
| `Order` | ☠️ No | accumulate | NO |
| `Joins` | ☠️ No | accumulate | NO |
| `Select` | ⚠️ No | overwrite | NO (but less dangerous) |
| `Distinct` | ⚠️ No | overwrite | NO (but less dangerous) |
| `Limit` | ✅ Yes (v1.25.7+) | - | Yes in v1.25.7+ |
| `Offset` | ✅ Yes (v1.25.7+) | - | Yes in v1.25.7+ |
| `Session` | ✅ Yes | - | Yes (but why would you?) |
| `Debug` | ✅ Yes | - | Yes |
| `WithContext` | ✅ Yes | - | Yes |
| `Begin` | ✅ Yes | - | Yes |
| `Create` | ✅ Yes | - | Yes |
| `Delete` | ✅ Yes (v1.21.8+) | - | Yes in v1.21.8+ |
| `Update` | ✅ Yes (v1.21.8+) | - | Yes in v1.21.8+ |
| `Find` | ☠️ No | - | Finisher - check results |

---

## Tools for Detection

Use [gormreuse](https://github.com/mpyw/gormreuse) linter to detect unsafe patterns at compile time:

```bash
go install github.com/mpyw/gormreuse/cmd/gormreuse@latest
gormreuse ./...
```

---

*Generated by [gorm-purity-survey](https://github.com/ryosuke.ishibashi/gorm-purity-survey)*
