# GORM Purity Survey Results

This document summarizes the method enumeration across all surveyed GORM versions.

## Overview

- **Versions surveyed**: 77
- **Version range**: v1.20.0 ~ v1.31.1

## Method Counts by Version

| Version | `*gorm.DB` Methods | Types | Pollution Paths |
|---------|-------------------|-------|-----------------|
| v1.20.0 | 70 | 12 | 106 |
| v1.20.1 | 70 | 12 | 106 |
| v1.20.2 | 70 | 12 | 106 |
| v1.20.3 | 70 | 12 | 106 |
| v1.20.4 | 70 | 12 | 106 |
| v1.20.5 | 70 | 12 | 106 |
| v1.20.6 | 70 | 12 | 106 |
| v1.20.7 | 71 | 12 | 108 |
| v1.20.8 | 71 | 12 | 108 |
| v1.20.9 | 71 | 12 | 108 |
| v1.20.10 | 71 | 12 | 108 |
| v1.20.11 | 71 | 12 | 108 |
| v1.20.12 | 71 | 12 | 108 |
| v1.21.0 | 73 | 12 | 110 |
| v1.21.1 | 73 | 12 | 110 |
| v1.21.2 | 73 | 12 | 110 |
| v1.21.3 | 73 | 12 | 110 |
| v1.21.4 | 73 | 12 | 110 |
| v1.21.5 | 73 | 12 | 110 |
| v1.21.6 | 73 | 12 | 110 |
| v1.21.7 | 73 | 12 | 110 |
| v1.21.8 | 73 | 12 | 110 |
| v1.21.9 | 73 | 12 | 110 |
| v1.21.10 | 73 | 12 | 111 |
| v1.21.11 | 73 | 12 | 111 |
| v1.21.12 | 73 | 12 | 111 |
| v1.21.13 | 73 | 12 | 111 |
| v1.21.14 | 73 | 12 | 111 |
| v1.21.15 | 73 | 12 | 111 |
| v1.21.16 | 73 | 12 | 111 |
| v1.22.0 | 73 | 12 | 111 |
| v1.22.1 | 73 | 12 | 111 |
| v1.22.2 | 73 | 12 | 111 |
| v1.22.3 | 74 | 12 | 113 |
| v1.22.4 | 74 | 12 | 113 |
| v1.22.5 | 75 | 12 | 115 |
| v1.23.0 | 75 | 12 | 115 |
| v1.23.1 | 75 | 12 | 115 |
| v1.23.2 | 75 | 12 | 115 |
| v1.23.3 | 75 | 12 | 115 |
| v1.23.4 | 75 | 12 | 115 |
| v1.23.5 | 75 | 12 | 115 |
| v1.23.6 | 75 | 12 | 115 |
| v1.23.7 | 75 | 12 | 115 |
| v1.23.8 | 75 | 12 | 115 |
| v1.23.9 | 75 | 12 | 115 |
| v1.23.10 | 75 | 12 | 115 |
| v1.24.0 | 75 | 12 | 115 |
| v1.24.1 | 75 | 12 | 115 |
| v1.24.2 | 75 | 12 | 115 |
| v1.24.3 | 76 | 12 | 117 |
| v1.24.4 | 76 | 12 | 117 |
| v1.24.5 | 76 | 12 | 117 |
| v1.24.6 | 76 | 12 | 117 |
| v1.25.0 | 76 | 13 | 117 |
| v1.25.1 | 76 | 14 | 117 |
| v1.25.2 | 76 | 14 | 117 |
| v1.25.3 | 76 | 14 | 117 |
| v1.25.4 | 76 | 14 | 117 |
| v1.25.5 | 76 | 14 | 117 |
| v1.25.6 | 76 | 14 | 117 |
| v1.25.7 | 76 | 14 | 117 |
| v1.25.8 | 76 | 14 | 117 |
| v1.25.9 | 76 | 14 | 117 |
| v1.25.10 | 76 | 14 | 117 |
| v1.25.11 | 77 | 14 | 119 |
| v1.25.12 | 77 | 14 | 119 |
| v1.26.0 | 77 | 14 | 119 |
| v1.26.1 | 77 | 14 | 119 |
| v1.30.0 | 77 | 16 | 119 |
| v1.30.1 | 77 | 16 | 119 |
| v1.30.2 | 77 | 16 | 119 |
| v1.30.3 | 77 | 16 | 119 |
| v1.30.4 | 77 | 16 | 119 |
| v1.30.5 | 77 | 16 | 119 |
| v1.31.0 | 77 | 16 | 119 |
| v1.31.1 | 77 | 16 | 119 |

## Method Count Changes

Versions where `*gorm.DB` method count changed from previous version:

- **v1.20.7**: 70 → 71 (+1 methods)
- **v1.21.0**: 71 → 73 (+2 methods)
- **v1.22.3**: 73 → 74 (+1 methods)
- **v1.22.5**: 74 → 75 (+1 methods)
- **v1.24.3**: 75 → 76 (+1 methods)
- **v1.25.11**: 76 → 77 (+1 methods)

## New Methods by Version

Methods added in each version (compared to immediate predecessor):

### v1.20.7

```
CreateInBatches
```

### v1.21.0

```
AfterInitialize
Apply
```

### v1.22.3

```
ToSQL
```

### v1.22.5

```
Connection
```

### v1.24.3

```
InnerJoins
```

### v1.25.11

```
MapColumns
```

## Generics API (v1.30+)

Starting from v1.30, GORM introduced Generics API with interfaces that hold internal `*gorm.DB`:

### v1.30.0

**PreloadBuilder** methods:

- `Limit() gorm.PreloadBuilder`
- `LimitPerRecord() gorm.PreloadBuilder`
- `Not(...interface {}) gorm.PreloadBuilder`
- `Offset() gorm.PreloadBuilder`
- `Omit() gorm.PreloadBuilder`
- `Or(...interface {}) gorm.PreloadBuilder`
- `Order() gorm.PreloadBuilder`
- `Select() gorm.PreloadBuilder`
- `Where(...interface {}) gorm.PreloadBuilder`

**JoinBuilder** methods:

- `Not(...interface {}) gorm.JoinBuilder`
- `Omit() gorm.JoinBuilder`
- `Or(...interface {}) gorm.JoinBuilder`
- `Select() gorm.JoinBuilder`
- `Where(...interface {}) gorm.JoinBuilder`

## Pollution Paths Summary

Methods that can potentially pollute `*gorm.DB` state:

### Chain Methods (return `*gorm.DB`)

- *gorm.DB.Assign returns *gorm.DB (chain point)
- *gorm.DB.Attrs returns *gorm.DB (chain point)
- *gorm.DB.Begin returns *gorm.DB (chain point)
- *gorm.DB.Clauses returns *gorm.DB (chain point)
- *gorm.DB.Commit returns *gorm.DB (chain point)
- *gorm.DB.Count returns *gorm.DB (chain point)
- *gorm.DB.Create returns *gorm.DB (chain point)
- *gorm.DB.CreateInBatches returns *gorm.DB (chain point)
- *gorm.DB.Debug returns *gorm.DB (chain point)
- *gorm.DB.Delete returns *gorm.DB (chain point)
- *gorm.DB.Distinct returns *gorm.DB (chain point)
- *gorm.DB.Exec returns *gorm.DB (chain point)
- *gorm.DB.Find returns *gorm.DB (chain point)
- *gorm.DB.FindInBatches returns *gorm.DB (chain point)
- *gorm.DB.First returns *gorm.DB (chain point)
- *gorm.DB.FirstOrCreate returns *gorm.DB (chain point)
- *gorm.DB.FirstOrInit returns *gorm.DB (chain point)
- *gorm.DB.Group returns *gorm.DB (chain point)
- *gorm.DB.Having returns *gorm.DB (chain point)
- *gorm.DB.InnerJoins returns *gorm.DB (chain point)

### Callback Methods (take `func(*gorm.DB)`)

- *gorm.DB.Connection takes func with *gorm.DB
- *gorm.DB.FindInBatches takes func with *gorm.DB
- *gorm.DB.ToSQL takes func with *gorm.DB
- *gorm.DB.Transaction takes func with *gorm.DB
- *gorm.Statement.Connection takes func with *gorm.DB
- *gorm.Statement.FindInBatches takes func with *gorm.DB
- *gorm.Statement.ToSQL takes func with *gorm.DB
- *gorm.Statement.Transaction takes func with *gorm.DB
- *gorm.callback.Register takes func with *gorm.DB
- *gorm.callback.Replace takes func with *gorm.DB
- *gorm.processor.Match takes func with *gorm.DB
- *gorm.processor.Register takes func with *gorm.DB
- *gorm.processor.Replace takes func with *gorm.DB

---

*Generated by gorm-purity-survey*
