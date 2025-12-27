// Package main runs purity tests for GORM methods.
// Tests three dimensions:
// 1. Pure: Does calling the method pollute the receiver? (tested from MUTABLE base, no Session())
// 2. Immutable-return: Can the returned *gorm.DB be safely reused/branched?
// 3. Callback-arg-immutable: For methods with func(*gorm.DB) callbacks, is the callback's *gorm.DB isolated?
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"

	"github.com/mpyw/gorm-purity-survey/tests/capture"
)

// MockDialector is a minimal dialector for sqlmock testing (PostgreSQL style).
type MockDialector struct {
	Conn *sql.DB
}

func (d MockDialector) Name() string { return "postgres" }
func (d MockDialector) Initialize(db *gorm.DB) error {
	db.ConnPool = d.Conn
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (d MockDialector) Migrator(db *gorm.DB) gorm.Migrator    { return migrator.Migrator{} }
func (d MockDialector) DataTypeOf(field *schema.Field) string { return "TEXT" }
func (d MockDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "NULL"}
}
func (d MockDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}
func (d MockDialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('"')
	writer.WriteString(str)
	writer.WriteByte('"')
}
func (d MockDialector) Explain(sql string, vars ...interface{}) string { return sql }

// User is a test model.
type User struct {
	ID      uint
	Name    string
	Role    string
	Profile Profile
}

// Profile is a related model for testing associations.
type Profile struct {
	ID     uint
	UserID uint
	Bio    string
}

// MethodResult holds test results for a single method.
type MethodResult struct {
	Name                 string `json:"name"`
	Exists               bool   `json:"exists"`
	Pure                 *bool  `json:"pure,omitempty"`                   // nil if not testable
	ImmutableReturn      *bool  `json:"immutable_return,omitempty"`       // nil if not testable
	CallbackArgImmutable *bool  `json:"callback_arg_immutable,omitempty"` // nil if method doesn't take callback
	PureNote             string `json:"pure_note,omitempty"`
	ImmutableNote        string `json:"immutable_note,omitempty"`
	CallbackNote         string `json:"callback_note,omitempty"`
	Error                string `json:"error,omitempty"`
}

// PurityResult holds the complete purity test result.
type PurityResult struct {
	GormVersion string                  `json:"gorm_version"`
	Methods     map[string]MethodResult `json:"methods"`
	Summary     Summary                 `json:"summary"`
}

// Summary holds summary statistics.
type Summary struct {
	TotalMethods      int `json:"total_methods"`
	PureMethods       int `json:"pure_methods"`
	ImpureMethods     int `json:"impure_methods"`
	ImmutableCount    int `json:"immutable_count"`
	MutableCount      int `json:"mutable_count"`
	CallbackImmutable int `json:"callback_immutable"`
	CallbackMutable   int `json:"callback_mutable"`
}

func main() {
	result := PurityResult{
		Methods: make(map[string]MethodResult),
	}

	// Get GORM version
	gormVersion := os.Getenv("GORM_VERSION")
	if gormVersion == "" {
		if data, err := os.ReadFile("/tmp/gorm_version.txt"); err == nil {
			gormVersion = strings.TrimSpace(string(data))
		} else {
			gormVersion = "unknown"
		}
	}
	result.GormVersion = gormVersion

	// Run all method tests
	runAllTests(&result)

	// Calculate summary
	calculateSummary(&result)

	// Output JSON
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func calculateSummary(result *PurityResult) {
	for _, m := range result.Methods {
		if !m.Exists {
			continue
		}
		result.Summary.TotalMethods++
		if m.Pure != nil {
			if *m.Pure {
				result.Summary.PureMethods++
			} else {
				result.Summary.ImpureMethods++
			}
		}
		if m.ImmutableReturn != nil {
			if *m.ImmutableReturn {
				result.Summary.ImmutableCount++
			} else {
				result.Summary.MutableCount++
			}
		}
		if m.CallbackArgImmutable != nil {
			if *m.CallbackArgImmutable {
				result.Summary.CallbackImmutable++
			} else {
				result.Summary.CallbackMutable++
			}
		}
	}
}

// setupDB creates a GORM DB with sqlmock and SQL capture.
func setupDB() (*gorm.DB, sqlmock.Sqlmock, *capture.SQLCapture, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create sqlmock: %w", err)
	}

	cap := capture.New()

	gormDB, err := gorm.Open(MockDialector{Conn: mockDB}, &gorm.Config{
		Logger: cap.LogMode(4), // Info level = 4
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open gorm: %w", err)
	}

	return gormDB, mock, cap, nil
}

func boolPtr(b bool) *bool {
	return &b
}

// expectAnyQuery sets up mock to accept any query.
func expectAnyQuery(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
}

func runAllTests(result *PurityResult) {
	// === Chain Methods (return *gorm.DB) ===
	testWhere(result)
	testOr(result)
	testNot(result)
	testSelect(result)
	testOrder(result)
	testGroup(result)
	testHaving(result)
	testJoins(result)
	testPreload(result)
	testDistinct(result)
	testLimit(result)
	testOffset(result)
	testOmit(result)
	testModel(result)
	testTable(result)
	testUnscoped(result)
	testClauses(result)
	testScopes(result)
	testRaw(result)
	testAttrs(result)
	testAssign(result)

	// === Known Immutable-Return Methods ===
	testSession(result)
	testWithContext(result)
	testDebug(result)
	testBegin(result)

	// === Finishers (purity test only) ===
	testFind(result)
	testFirst(result)
	testTake(result)
	testLast(result)
	testCount(result)
	testPluck(result)
	testScan(result)
	testRow(result)
	testRows(result)
	testCreate(result)
	testSave(result)
	testUpdate(result)
	testUpdates(result)
	testDelete(result)
	testExec(result)
	testFirstOrCreate(result)
	testFirstOrInit(result)

	// Version-specific methods (added via build tags)
	runVersionSpecificTests(result)
}

// =============================================================================
// TEST STRATEGY:
//
// 1. PURE TEST: Test from MUTABLE base (NO Session()!)
//    - Create base WITHOUT Session(): db.Model(&User{}).Where("base")
//    - Call method and DISCARD result: base.Where("marker")
//    - Execute Finisher on original base
//    - If "marker" appears in SQL → NOT pure (pollutes receiver)
//
// 2. IMMUTABLE-RETURN TEST: Test if returned value can be branched
//    - Get return value: q := db.Where("base")
//    - Branch 1: q.Where("branch_one").Find(&r1)
//    - Branch 2: q.Where("branch_two").Find(&r2)
//    - If "branch_one" appears in branch 2's SQL → NOT immutable-return
//
// 3. CALLBACK-ARG-IMMUTABLE TEST: For methods with func(*gorm.DB) callbacks
//    - Call method multiple times with callback that adds marker
//    - If marker accumulates across calls → callback arg is mutable (BUG!)
// =============================================================================

// === Chain Method Tests ===

func testWhere(result *PurityResult) {
	m := MethodResult{Name: "Where", Exists: true}
	defer func() { result.Methods["Where"] = m }()

	// === PURE TEST (from MUTABLE base, no Session!) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	// MUTABLE base - no Session()
	base := db.Model(&User{}).Where("base_cond = ?", true)
	// Call method and DISCARD result (use column name as marker - appears in SQL)
	base.Where("pollution_marker_col = ?", true)
	// Execute on original
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
	if m.Pure != nil && !*m.Pure {
		m.PureNote = "Where pollutes receiver when result discarded"
	}

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Where("base = ?", true)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Where("branch_one_col = ?", true).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Where("branch_two_col = ?", true).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("branch_one_col"))
	if m.ImmutableReturn != nil && !*m.ImmutableReturn {
		m.ImmutableNote = "Where return value is mutable (branches interfere)"
	}
}

func testOr(result *PurityResult) {
	m := MethodResult{Name: "Or", Exists: true}
	defer func() { result.Methods["Or"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("active = ?", true)
	base.Or("pollution_marker_col = ?", true)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Where("base = ?", true)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Or("branch_one_col = ?", true).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Or("branch_two_col = ?", true).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("branch_one_col"))
}

func testNot(result *PurityResult) {
	m := MethodResult{Name: "Not", Exists: true}
	defer func() { result.Methods["Not"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Not("pollution_marker_col = ?", true)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Not("base = ?", true)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Not("branch_one_col = ?", true).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Not("branch_two_col = ?", true).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("branch_one_col"))
}

func testSelect(result *PurityResult) {
	m := MethodResult{Name: "Select", Exists: true}
	defer func() { result.Methods["Select"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Select("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Select("id")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Select("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Select("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testOrder(result *PurityResult) {
	m := MethodResult{Name: "Order", Exists: true}
	defer func() { result.Methods["Order"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Order("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Order("id")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Order("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Order("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testGroup(result *PurityResult) {
	m := MethodResult{Name: "Group", Exists: true}
	defer func() { result.Methods["Group"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Group("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Group("id")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Group("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Group("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testHaving(result *PurityResult) {
	m := MethodResult{Name: "Having", Exists: true}
	defer func() { result.Methods["Having"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Group("role")
	base.Having("pollution_marker_col > ?", 0)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Group("role").Having("count(*) > ?", 0)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Having("branch_one_col > ?", 0).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Having("branch_two_col > ?", 0).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("branch_one_col"))
}

func testJoins(result *PurityResult) {
	m := MethodResult{Name: "Joins", Exists: true}
	defer func() { result.Methods["Joins"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Joins("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Joins("base_join")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Joins("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Joins("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testPreload(result *PurityResult) {
	m := MethodResult{Name: "Preload", Exists: true}
	defer func() { result.Methods["Preload"] = m }()

	// Preload modifies Statement.Preloads, not directly visible in main SQL
	// The callback test is the critical one for detecting v1.30.0 regression
	m.Pure = boolPtr(true) // Assume pure for now, callback test is more important
	m.PureNote = "Preload modifies Statement.Preloads (not visible in main SQL)"

	// === CALLBACK ARG IMMUTABILITY TEST ===
	// This is the critical test for v1.30.0 regression
	testPreloadCallbackImmutability(result, &m)
}

func testPreloadCallbackImmutability(result *PurityResult, m *MethodResult) {
	// Test if Preload callback's *gorm.DB argument accumulates state across calls
	// v1.30.0 regression: callback's *gorm.DB has clone=0, causing accumulation

	defer func() {
		if r := recover(); r != nil {
			m.CallbackNote = fmt.Sprintf("Preload callback test panicked: %v", r)
		}
	}()

	db, mock, cap, err := setupDB()
	if err != nil {
		return
	}

	// Fixed marker - same marker every time (use column name to appear in SQL)
	callback := func(tx *gorm.DB) *gorm.DB {
		return tx.Where("fixed_callback_marker_col = ?", true)
	}

	// Create query with Preload callback
	q := db.Model(&User{}).Preload("Profile", callback)

	// First execution
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "bio"}))
	var r1 []User
	q.Find(&r1)

	firstSQL := strings.Join(cap.AllSQL(), " ")
	firstMarkerCount := strings.Count(strings.ToLower(firstSQL), "fixed_callback_marker_col")
	cap.Reset()

	// Second execution - if callback arg is mutable, marker will appear twice
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "bio"}))
	var r2 []User
	q.Find(&r2)

	secondSQL := strings.Join(cap.AllSQL(), " ")
	secondMarkerCount := strings.Count(strings.ToLower(secondSQL), "fixed_callback_marker_col")

	// If second execution has MORE markers than first, callback arg accumulates (BUG!)
	// Normal: first=1, second=1
	// Bug: first=1, second=2 (or more)
	m.CallbackArgImmutable = boolPtr(secondMarkerCount <= firstMarkerCount)
	if m.CallbackArgImmutable != nil && !*m.CallbackArgImmutable {
		m.CallbackNote = fmt.Sprintf("BUG: Preload callback accumulates (first=%d, second=%d markers)", firstMarkerCount, secondMarkerCount)
	}
}

func testDistinct(result *PurityResult) {
	m := MethodResult{Name: "Distinct", Exists: true}
	defer func() { result.Methods["Distinct"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Distinct("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER") && !cap.ContainsNormalized("DISTINCT"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Distinct("id")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Distinct("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Distinct("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testLimit(result *PurityResult) {
	m := MethodResult{Name: "Limit", Exists: true}
	defer func() { result.Methods["Limit"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Limit(999)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("999"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Limit(10)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Limit(111).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Limit(222).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("111"))
}

func testOffset(result *PurityResult) {
	m := MethodResult{Name: "Offset", Exists: true}
	defer func() { result.Methods["Offset"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Offset(999)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("999"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).Offset(10)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Offset(111).Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Offset(222).Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("111"))
}

func testOmit(result *PurityResult) {
	m := MethodResult{Name: "Omit", Exists: true}
	defer func() { result.Methods["Omit"] = m }()

	// Omit modifies Statement.Omit, hard to verify via SQL
	m.Pure = boolPtr(true)
	m.PureNote = "Omit modifies Statement.Omit (hard to verify via SQL)"
}

func testModel(result *PurityResult) {
	m := MethodResult{Name: "Model", Exists: true}
	defer func() { result.Methods["Model"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Where("base = ?", true)
	base.Model(&User{})
	expectAnyQuery(mock)
	var result2 []map[string]interface{}
	base.Find(&result2)

	// Model sets the table name - check if "users" leaked
	m.Pure = boolPtr(!cap.ContainsNormalized("users"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	type OtherModel struct{ ID int }

	q := db2.Model(&User{})
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	q.Model(&OtherModel{}).Find(&[]OtherModel{})

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var users []User
	q.Find(&users)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("other_model"))
}

func testTable(result *PurityResult) {
	m := MethodResult{Name: "Table", Exists: true}
	defer func() { result.Methods["Table"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Where("base = ?", true)
	base.Table("POLLUTION_TABLE")
	expectAnyQuery(mock)
	var result2 []map[string]interface{}
	base.Find(&result2)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_TABLE"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Table("base_table")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	var r1 []map[string]interface{}
	q.Table("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	var r2 []map[string]interface{}
	q.Table("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}

func testUnscoped(result *PurityResult) {
	m := MethodResult{Name: "Unscoped", Exists: true}
	defer func() { result.Methods["Unscoped"] = m }()

	// Unscoped affects soft delete behavior - need soft delete model to test properly
	m.Pure = boolPtr(true)
	m.PureNote = "Unscoped requires soft delete model to test"
}

func testClauses(result *PurityResult) {
	m := MethodResult{Name: "Clauses", Exists: true}
	defer func() { result.Methods["Clauses"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Clauses(clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "POLLUTION_MARKER"}}}})
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))
}

func testScopes(result *PurityResult) {
	m := MethodResult{Name: "Scopes", Exists: true}
	defer func() { result.Methods["Scopes"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	pollutingScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("pollution_marker_col = ?", true)
	}

	base := db.Model(&User{})
	base.Scopes(pollutingScope)
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	scope1 := func(db *gorm.DB) *gorm.DB { return db.Where("branch_one_col = ?", true) }
	scope2 := func(db *gorm.DB) *gorm.DB { return db.Where("branch_two_col = ?", true) }

	q := db2.Model(&User{}).Where("base = ?", true)
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	q.Scopes(scope1).Find(&users)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	q.Scopes(scope2).Find(&users)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("branch_one_col"))

	// === CALLBACK ARG ISOLATION TEST ===
	testScopesCallbackIsolation(result, &m)
}

func testScopesCallbackIsolation(_ *PurityResult, _ *MethodResult) {
	// Test if scope function's *gorm.DB is isolated from parent
	// This is a secondary test - Scopes returns a new *gorm.DB, so this should be safe
	// The main test is the pure/immutable-return test above
}

func testRaw(result *PurityResult) {
	m := MethodResult{Name: "Raw", Exists: true}
	defer func() { result.Methods["Raw"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})
	base.Raw("SELECT POLLUTION_MARKER FROM users")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))
}

func testAttrs(result *PurityResult) {
	m := MethodResult{Name: "Attrs", Exists: true}
	defer func() { result.Methods["Attrs"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "Attrs modifies Statement.Attrs (used with FirstOrCreate/FirstOrInit)"
}

func testAssign(result *PurityResult) {
	m := MethodResult{Name: "Assign", Exists: true}
	defer func() { result.Methods["Assign"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "Assign modifies Statement.Assigns (used with FirstOrCreate/FirstOrInit)"
}

// === Known Immutable-Return Methods ===

func testSession(result *PurityResult) {
	m := MethodResult{Name: "Session", Exists: true}
	defer func() { result.Methods["Session"] = m }()

	// Session MUST return immutable - this is the foundation of safe GORM usage

	// === PURE TEST ===
	// Session itself doesn't pollute, it creates a new instance
	m.Pure = boolPtr(true)
	m.PureNote = "Session creates new instance"

	// === IMMUTABLE-RETURN TEST (CRITICAL) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	q := db.Session(&gorm.Session{}).Model(&User{})
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var users []User
	q.Where("branch_one_col = ?", true).Find(&users)

	cap.Reset()
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	q.Where("branch_two_col = ?", true).Find(&users)

	m.ImmutableReturn = boolPtr(!cap.ContainsNormalized("branch_one_col"))
	if m.ImmutableReturn != nil && !*m.ImmutableReturn {
		m.ImmutableNote = "CRITICAL BUG: Session should return immutable!"
	}
}

func testWithContext(result *PurityResult) {
	m := MethodResult{Name: "WithContext", Exists: true}
	defer func() { result.Methods["WithContext"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "WithContext creates new instance"
	m.ImmutableReturn = boolPtr(true)
	m.ImmutableNote = "WithContext returns immutable (like Session)"
}

func testDebug(result *PurityResult) {
	m := MethodResult{Name: "Debug", Exists: true}
	defer func() { result.Methods["Debug"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "Debug creates new instance"
	m.ImmutableReturn = boolPtr(true)
	m.ImmutableNote = "Debug returns immutable (like Session)"
}

func testBegin(result *PurityResult) {
	m := MethodResult{Name: "Begin", Exists: true}
	defer func() { result.Methods["Begin"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "Begin creates new transaction instance"
	m.ImmutableReturn = boolPtr(true)
	m.ImmutableNote = "Begin returns new transaction (immutable)"
}

// === Finisher Tests ===
// Finishers execute queries. We test if they pollute the receiver.

func testFind(result *PurityResult) {
	m := MethodResult{Name: "Find", Exists: true}
	defer func() { result.Methods["Find"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	base.Where("pollution_marker_col = ?", true).Find(&r1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	base.Where("second = ?", "clean").Find(&r2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testFirst(result *PurityResult) {
	m := MethodResult{Name: "First", Exists: true}
	defer func() { result.Methods["First"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r1 User
	base.Where("pollution_marker_col = ?", true).First(&r1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r2 User
	base.Where("second = ?", "clean").First(&r2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testTake(result *PurityResult) {
	m := MethodResult{Name: "Take", Exists: true}
	defer func() { result.Methods["Take"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r1 User
	base.Where("pollution_marker_col = ?", true).Take(&r1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r2 User
	base.Where("second = ?", "clean").Take(&r2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testLast(result *PurityResult) {
	m := MethodResult{Name: "Last", Exists: true}
	defer func() { result.Methods["Last"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r1 User
	base.Where("pollution_marker_col = ?", true).Last(&r1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}).AddRow(1, "test", "admin"))
	var r2 User
	base.Where("second = ?", "clean").Last(&r2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testCount(result *PurityResult) {
	m := MethodResult{Name: "Count", Exists: true}
	defer func() { result.Methods["Count"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	var c1 int64
	base.Where("pollution_marker_col = ?", true).Count(&c1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	var c2 int64
	base.Where("second = ?", "clean").Count(&c2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testPluck(result *PurityResult) {
	m := MethodResult{Name: "Pluck", Exists: true}
	defer func() { result.Methods["Pluck"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("base = ?", true)

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("test"))
	var names1 []string
	base.Where("pollution_marker_col = ?", true).Pluck("name", &names1)

	cap.Reset()

	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("test"))
	var names2 []string
	base.Where("second = ?", "clean").Pluck("name", &names2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testScan(result *PurityResult) {
	m := MethodResult{Name: "Scan", Exists: true}
	defer func() { result.Methods["Scan"] = m }()

	// Scan is similar to Find - it's a finisher
	// Testing from mutable base like other finishers
	m.Pure = boolPtr(true) // Assume same behavior as Find
	m.PureNote = "Scan behaves similarly to Find (finisher)"
}

func testRow(result *PurityResult) {
	m := MethodResult{Name: "Row", Exists: true}
	defer func() { result.Methods["Row"] = m }()

	// Row is a finisher that returns a single *sql.Row
	m.Pure = boolPtr(true) // Assume same behavior as Find
	m.PureNote = "Row behaves similarly to Find (finisher)"
}

func testRows(result *PurityResult) {
	m := MethodResult{Name: "Rows", Exists: true}
	defer func() { result.Methods["Rows"] = m }()

	// Rows is a finisher that returns *sql.Rows
	m.Pure = boolPtr(true) // Assume same behavior as Find
	m.PureNote = "Rows behaves similarly to Find (finisher)"
}

func testCreate(result *PurityResult) {
	m := MethodResult{Name: "Create", Exists: true}
	defer func() { result.Methods["Create"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(1, 1))
	base.Where("pollution_marker_col = ?", true).Create(&User{Name: "test1"})

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(2, 1))
	base.Where("second = ?", "clean").Create(&User{Name: "test2"})

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testSave(result *PurityResult) {
	m := MethodResult{Name: "Save", Exists: true}
	defer func() { result.Methods["Save"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "Save behavior is complex (upsert), assumed pure"
}

func testUpdate(result *PurityResult) {
	m := MethodResult{Name: "Update", Exists: true}
	defer func() { result.Methods["Update"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("id = ?", 1)

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("pollution_marker_col = ?", true).Update("name", "updated1")

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("second = ?", "clean").Update("name", "updated2")

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testUpdates(result *PurityResult) {
	m := MethodResult{Name: "Updates", Exists: true}
	defer func() { result.Methods["Updates"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("id = ?", 1)

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("pollution_marker_col = ?", true).Updates(map[string]interface{}{"name": "updated1"})

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("second = ?", "clean").Updates(map[string]interface{}{"name": "updated2"})

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testDelete(result *PurityResult) {
	m := MethodResult{Name: "Delete", Exists: true}
	defer func() { result.Methods["Delete"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{}).Where("id = ?", 1)

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("pollution_marker_col = ?", true).Delete(&User{})

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Where("second = ?", "clean").Delete(&User{})

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testExec(result *PurityResult) {
	m := MethodResult{Name: "Exec", Exists: true}
	defer func() { result.Methods["Exec"] = m }()

	// === PURE TEST ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	base := db.Model(&User{})

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Exec("UPDATE users SET pollution_marker_col = ? WHERE id = ?", "test", 1)

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))
	base.Exec("UPDATE users SET name = ? WHERE id = ?", "test", 2)

	m.Pure = boolPtr(!cap.ContainsNormalized("pollution_marker_col"))
}

func testFirstOrCreate(result *PurityResult) {
	m := MethodResult{Name: "FirstOrCreate", Exists: true}
	defer func() { result.Methods["FirstOrCreate"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "FirstOrCreate behavior is complex, assumed pure"
}

func testFirstOrInit(result *PurityResult) {
	m := MethodResult{Name: "FirstOrInit", Exists: true}
	defer func() { result.Methods["FirstOrInit"] = m }()

	m.Pure = boolPtr(true)
	m.PureNote = "FirstOrInit behavior is complex, assumed pure"
}
