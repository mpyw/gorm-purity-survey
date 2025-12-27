//go:build gorm_v125plus

package main

import (
	"github.com/DATA-DOG/go-sqlmock"
)

// runV125Tests tests methods added in v1.25+
func runV125Tests(result *PurityResult) {
	testInnerJoins(result)

	// v1.26+ methods
	runV126Tests(result)
}

func testInnerJoins(result *PurityResult) {
	m := MethodResult{Name: "InnerJoins", Exists: true}
	defer func() { result.Methods["InnerJoins"] = m }()

	// === PURE TEST (from MUTABLE base, no Session!) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	// MUTABLE base - no Session()
	base := db.Model(&User{})
	base.InnerJoins("POLLUTION_MARKER")
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))

	// === IMMUTABLE-RETURN TEST ===
	db2, mock2, cap2, err := setupDB()
	if err != nil {
		return
	}

	q := db2.Model(&User{}).InnerJoins("base_join")
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r1 []User
	q.InnerJoins("BRANCH_ONE").Find(&r1)

	cap2.Reset()
	mock2.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "role"}))
	var r2 []User
	q.InnerJoins("BRANCH_TWO").Find(&r2)

	m.ImmutableReturn = boolPtr(!cap2.ContainsNormalized("BRANCH_ONE"))
}
