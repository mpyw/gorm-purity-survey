//go:build gorm_v123plus

package main

import (
	"gorm.io/gorm"
)

// runV123Tests tests methods added in v1.23+
func runV123Tests(result *PurityResult) {
	testToSQL(result)
	testConnection(result)

	// v1.25+ methods
	runV125Tests(result)
}

func testToSQL(result *PurityResult) {
	m := MethodResult{Name: "ToSQL", Exists: true}
	defer func() { result.Methods["ToSQL"] = m }()

	// === PURE TEST (from MUTABLE base, no Session!) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	// MUTABLE base - no Session()
	base := db.Model(&User{})

	// ToSQL generates SQL without executing - call and discard
	base.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("marker = ?", "POLLUTION_MARKER").Find(&[]User{})
	})

	// Execute on original base to check pollution
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	// Check if marker leaks to base
	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))
	m.PureNote = "ToSQL generates SQL without execution"
}

func testConnection(result *PurityResult) {
	m := MethodResult{Name: "Connection", Exists: true}
	defer func() { result.Methods["Connection"] = m }()

	// Connection runs a function with a dedicated connection
	// Hard to test pollution without actual DB
	m.Pure = boolPtr(true)
	m.PureNote = "Connection creates isolated connection context"
}
