//go:build gorm_v126plus

package main

// runV126Tests tests methods added in v1.26+
func runV126Tests(result *PurityResult) {
	testMapColumns(result)
}

func testMapColumns(result *PurityResult) {
	m := MethodResult{Name: "MapColumns", Exists: true}
	defer func() { result.Methods["MapColumns"] = m }()

	// === PURE TEST (from MUTABLE base, no Session!) ===
	db, mock, cap, err := setupDB()
	if err != nil {
		m.Error = err.Error()
		return
	}

	// MUTABLE base - no Session()
	base := db.Model(&User{})
	// MapColumns maps column names - call and discard result
	base.MapColumns(map[string]string{"name": "POLLUTION_MARKER"})
	expectAnyQuery(mock)
	var users []User
	base.Find(&users)

	m.Pure = boolPtr(!cap.ContainsNormalized("POLLUTION_MARKER"))
	m.PureNote = "MapColumns modifies column mapping"
}
