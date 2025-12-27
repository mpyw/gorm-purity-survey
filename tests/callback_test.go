// Package tests provides purity survey tests for GORM methods.
package tests

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

// === Callback Method Tests ===
//
// These test methods that receive func(*gorm.DB) callbacks:
// - Scopes(...func(*gorm.DB) *gorm.DB)
// - Transaction(func(*gorm.DB) error)
// - Connection(func(*gorm.DB) error)
// - FindInBatches(dest, batchSize, func(*gorm.DB, int) error)
// - ToSQL(func(*gorm.DB) *gorm.DB)
// - Preload(relation, ...interface{}) - can receive func(*gorm.DB) *gorm.DB
//
// Key questions:
// 1. Is the *gorm.DB passed to callback isolated from the parent?
// 2. Do changes in the callback leak to the parent?
// 3. Do changes to the parent leak into the callback?

// TestCallback_Scopes_Isolation tests if Scopes callback receives isolated DB.
func TestCallback_Scopes_Isolation(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{})

	// Scope that adds a condition
	addAdmin := func(db *gorm.DB) *gorm.DB {
		return db.Where("role = ?", "admin")
	}

	// Apply scope
	scoped := base.Scopes(addAdmin)

	// First query with scope
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	scoped.Find(&r1)

	cap.Reset()

	// Query on original base - should NOT have "admin"
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	base.Find(&r2)

	if cap.ContainsNormalized("admin") {
		t.Log("Scopes LEAKS to parent (callback not isolated)")
	} else {
		t.Log("Scopes does NOT leak to parent (callback isolated)")
	}
}

// TestCallback_Scopes_MultipleScopes tests if multiple scopes interfere.
func TestCallback_Scopes_MultipleScopes(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{})

	scope1 := func(db *gorm.DB) *gorm.DB {
		return db.Where("scope1 = ?", true)
	}
	scope2 := func(db *gorm.DB) *gorm.DB {
		return db.Where("scope2 = ?", true)
	}

	// Apply scope1 first
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	base.Scopes(scope1).Find(&r1)

	cap.Reset()

	// Apply scope2 - should NOT contain scope1
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	base.Scopes(scope2).Find(&r2)

	if cap.ContainsNormalized("scope1") {
		t.Log("Scopes calls INTERFERE with each other")
	} else {
		t.Log("Scopes calls are independent")
	}
}

// TestCallback_Scopes_MutatesCallback tests if scope callback can mutate outside state.
func TestCallback_Scopes_MutatesCallback(t *testing.T) {
	db, mock, cap := setupDB(t)

	// Will the callback's db be the same instance as base?
	var callbackDB *gorm.DB

	scope := func(db *gorm.DB) *gorm.DB {
		callbackDB = db
		return db.Where("in_callback = ?", true)
	}

	base := db.Session(&gorm.Session{}).Model(&User{})

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var users []User
	base.Scopes(scope).Find(&users)

	// Check if callbackDB is the same pointer as base
	// This would indicate shared state
	if callbackDB == base {
		t.Log("WARNING: Scope callback receives SAME instance as base (dangerous)")
	} else {
		t.Log("Scope callback receives DIFFERENT instance from base")
	}
	_ = cap // unused but available for SQL checks
}

// TestCallback_Preload_FuncCallback tests Preload with function callback.
func TestCallback_Preload_FuncCallback(t *testing.T) {
	db, mock, cap := setupDB(t)

	type Order struct {
		ID     uint
		UserID uint
	}

	type UserWithOrders struct {
		ID     uint
		Name   string
		Orders []Order
	}

	base := db.Session(&gorm.Session{}).Model(&UserWithOrders{})

	// Preload with callback that adds condition
	preloadCallback := func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", "active")
	}

	// Note: This test may not work perfectly with sqlmock
	// because Preload issues a separate query
	mock.ExpectQuery(".*users.*").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))
	mock.ExpectQuery(".*orders.*").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id"}))

	var users []UserWithOrders
	base.Preload("Orders", preloadCallback).Find(&users)

	// Check if callback condition appears in the right query
	allSQL := cap.AllSQL()
	t.Logf("Captured SQL: %v", allSQL)

	// The callback should only affect the Preload query, not the main query
	for i, sql := range allSQL {
		if cap.ContainsNormalized("status") {
			t.Logf("Query %d contains callback condition: %s", i, sql)
		}
	}
}

// TestCallback_Transaction_Isolation tests if Transaction callback is isolated.
func TestCallback_Transaction_Isolation(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{}).Where("base = ?", true)

	// Transaction callback
	var txDB *gorm.DB
	txFunc := func(tx *gorm.DB) error {
		txDB = tx
		// Add condition inside transaction
		tx.Where("in_tx = ?", true)
		return nil
	}

	// Setup transaction expectations
	mock.ExpectBegin()
	mock.ExpectCommit()

	err := base.Transaction(txFunc)
	if err != nil {
		t.Logf("Transaction error (may be expected with mock): %v", err)
	}

	cap.Reset()

	// Query on original base - should NOT have "in_tx"
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var users []User
	base.Find(&users)

	if cap.ContainsNormalized("in_tx") {
		t.Log("Transaction callback LEAKS to parent")
	} else {
		t.Log("Transaction callback is isolated from parent")
	}

	// Check if tx is same as base
	if txDB == base {
		t.Log("WARNING: Transaction receives SAME instance (dangerous)")
	}
}

// TestCallback_FindInBatches_Isolation tests FindInBatches callback isolation.
func TestCallback_FindInBatches_Isolation(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{}).Where("base = ?", true)

	var batchDBs []*gorm.DB

	// Setup mock to return some rows for batching
	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name", "role"}).
			AddRow(1, "user1", "admin").
			AddRow(2, "user2", "user"),
	)

	var allUsers []User
	err := base.FindInBatches(&allUsers, 10, func(tx *gorm.DB, batch int) error {
		batchDBs = append(batchDBs, tx)
		// Try to mutate
		tx.Where("mutated_in_batch = ?", batch)
		return nil
	}).Error

	if err != nil {
		t.Logf("FindInBatches error: %v", err)
	}

	cap.Reset()

	// Query on original base
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var users []User
	base.Find(&users)

	if cap.ContainsNormalized("mutated_in_batch") {
		t.Log("FindInBatches callback LEAKS to parent")
	} else {
		t.Log("FindInBatches callback is isolated from parent")
	}
}

// TestCallback_ToSQL_Isolation tests ToSQL callback isolation.
func TestCallback_ToSQL_Isolation(t *testing.T) {
	db, _, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{}).Where("base = ?", true)

	// ToSQL callback
	sql := base.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("in_tosql = ?", true).Find(&[]User{})
	})

	t.Logf("ToSQL result: %s", sql)

	cap.Reset()

	// The base should not be affected
	// Note: ToSQL doesn't execute, so we check differently
	if base.Statement != nil && base.Statement.SQL.String() != "" {
		// This check may not work as expected
		t.Logf("Base statement after ToSQL: %s", base.Statement.SQL.String())
	}
}

// === Summary Test: All Callback Methods ===

// CallbackTestResult holds result of callback isolation test.
type CallbackTestResult struct {
	Method           string
	CallbackIsolated bool // Callback changes don't leak to parent
	ParentIsolated   bool // Parent changes don't leak to callback
	SameInstance     bool // Callback receives same *gorm.DB pointer (dangerous)
	Notes            string
}

// TestCallback_Summary runs all callback tests and summarizes.
func TestCallback_Summary(t *testing.T) {
	results := []CallbackTestResult{
		{Method: "Scopes", Notes: "See individual tests above"},
		{Method: "Transaction", Notes: "See individual tests above"},
		{Method: "Connection", Notes: "Similar to Transaction"},
		{Method: "FindInBatches", Notes: "See individual tests above"},
		{Method: "ToSQL", Notes: "Special case - doesn't execute SQL"},
		{Method: "Preload (func)", Notes: "Applies to Preload query only"},
	}

	t.Log("=== Callback Method Isolation Summary ===")
	for _, r := range results {
		t.Logf("%s: %s", r.Method, r.Notes)
	}
}
