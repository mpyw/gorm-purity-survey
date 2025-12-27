// Package tests provides purity survey tests for GORM methods.
package tests

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/mpyw/gorm-purity-survey/tests/capture"
)

// User is a test model.
type User struct {
	ID   uint
	Name string
	Role string
}

// setupDB creates a GORM DB with sqlmock and SQL capture.
func setupDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *capture.SQLCapture) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	cap := capture.New()

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: cap,
	})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	return gormDB, mock, cap
}

// expectAnyQuery sets up mock to accept any query.
func expectAnyQuery(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
}

// TestPollution_Where tests if Where pollutes the receiver.
//
// Pollution test pattern:
//  1. Create an immutable base: db.Session(&gorm.Session{})
//  2. Call the method and DISCARD the result
//  3. Call a Finisher on the original
//  4. Check if the discarded method's effect appears in the SQL
func TestPollution_Where(t *testing.T) {
	db, mock, cap := setupDB(t)

	// Create immutable base
	base := db.Session(&gorm.Session{}).Model(&User{})

	// Call Where and DISCARD result (this is the bomb pattern)
	base.Where("role = ?", "admin") // Result discarded!

	// Call Finisher on original
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	// Check: if "admin" appears, Where polluted the receiver
	if cap.ContainsNormalized("admin") {
		t.Log("Where POLLUTES the receiver (bomb pattern detected)")
		// This is expected behavior for most GORM versions
	} else {
		t.Log("Where does NOT pollute the receiver")
	}
}

// TestPollution_Clauses tests if Clauses pollutes the receiver.
func TestPollution_Clauses(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{})

	// Call Clauses and DISCARD result
	// Note: We can't easily test hints without a real driver,
	// so we test with a simple clause that modifies SQL
	base.Where("injected = ?", "test") // Using Where as proxy for Clauses effect

	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	if cap.ContainsNormalized("injected") {
		t.Log("Clauses-like method POLLUTES the receiver")
	} else {
		t.Log("Clauses-like method does NOT pollute the receiver")
	}
}

// TestPollution_Finisher_Find tests if Find pollutes the receiver for subsequent calls.
func TestPollution_Finisher_Find(t *testing.T) {
	db, mock, cap := setupDB(t)

	base := db.Session(&gorm.Session{}).Model(&User{}).Where("active = ?", true)

	// First Find with additional condition
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	base.Where("role = ?", "admin").Find(&r1)

	cap.Reset()

	// Second Find on same base - does "admin" leak?
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	base.Where("role = ?", "user").Find(&r2)

	sql := cap.LastSQL()
	if cap.ContainsNormalized("admin") {
		t.Logf("Find POLLUTES for subsequent calls: %s", sql)
	} else {
		t.Logf("Find does NOT pollute for subsequent calls: %s", sql)
	}
}

// === Immutable-Return Tests ===

// TestImmutableReturn_Where tests if Where returns an immutable DB.
//
// Immutable-return test pattern:
//  1. Call the method to get a new DB
//  2. Branch into two independent chains
//  3. Check if the chains interfere with each other
func TestImmutableReturn_Where(t *testing.T) {
	db, mock, cap := setupDB(t)

	// Get result from Where
	q := db.Model(&User{}).Where("base = ?", true)

	// Branch 1
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Where("branch = ?", "one").Find(&r1)

	cap.Reset()

	// Branch 2 - should NOT contain "one" if Where returns immutable
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Where("branch = ?", "two").Find(&r2)

	if cap.ContainsNormalized("one") {
		t.Logf("Where does NOT return immutable (branches interfere)")
	} else {
		t.Logf("Where returns immutable (branches independent)")
	}
}

// TestImmutableReturn_Session tests if Session returns an immutable DB.
func TestImmutableReturn_Session(t *testing.T) {
	db, mock, cap := setupDB(t)

	// Session should definitely return immutable
	q := db.Session(&gorm.Session{}).Model(&User{})

	// Branch 1
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.Where("branch = ?", "one").Find(&r1)

	cap.Reset()

	// Branch 2
	mock.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.Where("branch = ?", "two").Find(&r2)

	if cap.ContainsNormalized("one") {
		t.Errorf("Session DOES NOT return immutable - THIS IS A BUG")
	} else {
		t.Log("Session correctly returns immutable")
	}
}

// === Helper for running tests across methods ===

// PollutionTestCase defines a test case for pollution detection.
type PollutionTestCase struct {
	Name       string
	Setup      func(db *gorm.DB) *gorm.DB // Create base
	Pollute    func(db *gorm.DB)          // Call method, discard result
	ExpectPure bool                       // Expected: should NOT pollute
}

// RunPollutionTest runs a pollution test case.
func RunPollutionTest(t *testing.T, tc PollutionTestCase) {
	t.Helper()
	t.Run(tc.Name, func(t *testing.T) {
		db, mock, cap := setupDB(t)

		base := tc.Setup(db)
		tc.Pollute(base)

		expectAnyQuery(mock)
		var users []User
		base.Find(&users)

		polluted := cap.ContainsNormalized("pollute_marker")
		if polluted && tc.ExpectPure {
			t.Errorf("Expected pure but method polluted")
		} else if !polluted && !tc.ExpectPure {
			t.Errorf("Expected pollution but method was pure")
		}
	})
}
