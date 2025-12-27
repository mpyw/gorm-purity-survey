//go:build gorm_v121plus

package main

import (
	"github.com/DATA-DOG/go-sqlmock"
)

// runVersionSpecificTests tests methods added in v1.21+
func runVersionSpecificTests(result *PurityResult) {
	testCreateInBatches(result)

	// v1.23+ methods
	runV123Tests(result)
}

func testCreateInBatches(result *PurityResult) {
	m := MethodResult{Name: "CreateInBatches", Exists: true}
	defer func() { result.Methods["CreateInBatches"] = m }()

	// === PURE TEST (from MUTABLE base, no Session!) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	// MUTABLE base - no Session()
	base := db.Model(&User{})

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(1, 2))
	users := []User{{Name: "test1"}, {Name: "test2"}}
	base.Where("marker = ?", "POLLUTION_MARKER").CreateInBatches(&users, 10)

	cap.Reset()

	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(3, 2))
	users2 := []User{{Name: "test3"}, {Name: "test4"}}
	base.Where("second = ?", "clean").CreateInBatches(&users2, 10)

	m.Pure = ptr(!cap.ContainsNormalized("POLLUTION_MARKER"))
}
