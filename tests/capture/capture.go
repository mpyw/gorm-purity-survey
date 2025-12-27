// Package capture provides SQL capture utilities for GORM purity testing.
package capture

import (
	"context"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"
)

// SQLCapture captures SQL statements executed by GORM.
// Implements gorm/logger.Interface.
type SQLCapture struct {
	mu      sync.Mutex
	Queries []CapturedQuery
	silent  bool
}

// CapturedQuery holds a captured SQL query with metadata.
type CapturedQuery struct {
	SQL          string
	RowsAffected int64
	Error        error
	Duration     time.Duration
}

// New creates a new SQLCapture logger.
func New() *SQLCapture {
	return &SQLCapture{}
}

// Reset clears all captured queries.
func (c *SQLCapture) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Queries = nil
}

// LastSQL returns the last captured SQL, or empty string if none.
func (c *SQLCapture) LastSQL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.Queries) == 0 {
		return ""
	}
	return c.Queries[len(c.Queries)-1].SQL
}

// AllSQL returns all captured SQL statements.
func (c *SQLCapture) AllSQL() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]string, len(c.Queries))
	for i, q := range c.Queries {
		result[i] = q.SQL
	}
	return result
}

// Contains checks if any captured SQL contains the substring.
func (c *SQLCapture) Contains(substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, q := range c.Queries {
		if strings.Contains(q.SQL, substr) {
			return true
		}
	}
	return false
}

// ContainsNormalized checks if any captured SQL contains the substring (case-insensitive, whitespace-normalized).
func (c *SQLCapture) ContainsNormalized(substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	normSubstr := normalize(substr)
	for _, q := range c.Queries {
		if strings.Contains(normalize(q.SQL), normSubstr) {
			return true
		}
	}
	return false
}

// normalize normalizes SQL for comparison (lowercase, collapse whitespace).
func normalize(sql string) string {
	sql = strings.ToLower(sql)
	// Collapse multiple whitespace to single space
	fields := strings.Fields(sql)
	return strings.Join(fields, " ")
}

// === gorm/logger.Interface implementation ===

// LogMode returns the logger with the specified log level.
func (c *SQLCapture) LogMode(level logger.LogLevel) logger.Interface {
	c.silent = level == logger.Silent
	return c
}

// Info logs info level messages (ignored for capture).
func (c *SQLCapture) Info(_ context.Context, _ string, _ ...interface{}) {}

// Warn logs warn level messages (ignored for capture).
func (c *SQLCapture) Warn(_ context.Context, _ string, _ ...interface{}) {}

// Error logs error level messages (ignored for capture).
func (c *SQLCapture) Error(_ context.Context, _ string, _ ...interface{}) {}

// Trace logs SQL statements - this is where we capture.
func (c *SQLCapture) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if c.silent {
		return
	}
	sql, rows := fc()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Queries = append(c.Queries, CapturedQuery{
		SQL:          sql,
		RowsAffected: rows,
		Error:        err,
		Duration:     time.Since(begin),
	})
}
