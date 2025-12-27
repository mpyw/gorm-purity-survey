// Package main explores gorm.Statement fields using godump
// to identify which fields are relevant for pollution detection.
package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/goforj/godump"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// MockDialector is a minimal dialector for sqlmock testing.
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
	ID   uint
	Name string
	Role string
}

func setupDB() (*gorm.DB, error) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	return gorm.Open(MockDialector{Conn: sqlDB}, &gorm.Config{
		SkipDefaultTransaction: true,
	})
}

func setupDBWithMock() (*gorm.DB, sqlmock.Sqlmock, *sql.DB, error) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, nil, err
	}
	db, err := gorm.Open(MockDialector{Conn: sqlDB}, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	return db, mock, sqlDB, err
}

// snapshotRelevantFields captures pollution-relevant Statement fields
type snapshotRelevantFields struct {
	ClauseCount  int
	ClauseKeys   []string
	Selects      []string
	Omits        []string
	JoinsLen     int
	PreloadsLen  int
	Distinct     bool
	Table        string
	WhereExprs   int // count of WHERE expressions
}

func snapshotStatement(stmt *gorm.Statement) snapshotRelevantFields {
	s := snapshotRelevantFields{
		ClauseCount: len(stmt.Clauses),
		Selects:     stmt.Selects,
		Omits:       stmt.Omits,
		JoinsLen:    len(stmt.Joins),
		Distinct:    stmt.Distinct,
		Table:       stmt.Table,
	}

	// Get clause keys
	for k := range stmt.Clauses {
		s.ClauseKeys = append(s.ClauseKeys, k)
	}

	// Get preloads count using reflection (since Preloads might not exist in all versions)
	rv := reflect.ValueOf(stmt).Elem()
	if preloads := rv.FieldByName("Preloads"); preloads.IsValid() && !preloads.IsNil() {
		s.PreloadsLen = preloads.Len()
	}

	// Count WHERE expressions
	if whereClause, ok := stmt.Clauses["WHERE"]; ok {
		if whereExpr, ok := whereClause.Expression.(clause.Where); ok {
			s.WhereExprs = len(whereExpr.Exprs)
		}
	}

	return s
}

// Create a focused dumper for Statement comparison
// Only include pollution-relevant fields
var statementDumper = godump.NewDumper(
	godump.WithMaxDepth(5),
	godump.WithOnlyFields(
		"Clauses",       // WHERE, ORDER BY, LIMIT, OFFSET, GROUP BY, HAVING, etc.
		"Joins",         // JOIN information
		"Preloads",      // Preload configurations
		"Selects",       // SELECT columns
		"Omits",         // OMIT columns
		"Table",         // Table name
		"Distinct",      // DISTINCT flag
		"ColumnMapping", // Column mapping
	),
)

// dbDumper focuses on *gorm.DB's clone field (unexported but critical!)
var dbDumper = godump.NewDumper(
	godump.WithMaxDepth(2),
	godump.WithOnlyFields(
		"clone", // CRITICAL: 0=no clone, 1=clone Statement, 2=clone DB
	),
)

// isPolluted uses godump to detect if Statement changed
func isPolluted(before, after *gorm.Statement) bool {
	beforeJSON := statementDumper.DumpJSONStr(before)
	afterJSON := statementDumper.DumpJSONStr(after)
	return beforeJSON != afterJSON
}

// getPollutionDiff uses godump to show what changed
func getPollutionDiff(before, after *gorm.Statement) string {
	return godump.DiffStr(before, after)
}

// getCloneValue extracts the unexported clone field from *gorm.DB
func getCloneValue(db *gorm.DB) int64 {
	rv := reflect.ValueOf(db).Elem()
	cloneField := rv.FieldByName("clone")
	if !cloneField.IsValid() {
		return -1 // field doesn't exist
	}
	return cloneField.Int()
}

func main() {
	fmt.Println("=== CLONE FIELD SURVEY (immutable-return detection) ===\n")

	// Survey all chain methods for their clone values
	fmt.Println("Method                  | Clone | Immutable-Return")
	fmt.Println("------------------------|-------|------------------")

	db, _ := setupDB()

	// Base DB
	fmt.Printf("%-23s | %5d | %s\n", "Fresh DB", getCloneValue(db), cloneToImmutable(getCloneValue(db)))

	// Session-like methods
	fmt.Printf("%-23s | %5d | %s\n", "Session()", getCloneValue(db.Session(&gorm.Session{})), cloneToImmutable(getCloneValue(db.Session(&gorm.Session{}))))
	// Session(NewDB:true) - check if NewDB field exists
	sessionType := reflect.TypeOf(gorm.Session{})
	if _, hasNewDB := sessionType.FieldByName("NewDB"); hasNewDB {
		session := &gorm.Session{}
		reflect.ValueOf(session).Elem().FieldByName("NewDB").SetBool(true)
		fmt.Printf("%-23s | %5d | %s\n", "Session(NewDB:true)", getCloneValue(db.Session(session)), cloneToImmutable(getCloneValue(db.Session(session))))
	} else {
		fmt.Printf("%-23s | %5s | %s\n", "Session(NewDB:true)", "N/A", "(NewDB not available)")
	}
	fmt.Printf("%-23s | %5d | %s\n", "WithContext()", getCloneValue(db.WithContext(db.Statement.Context)), cloneToImmutable(getCloneValue(db.WithContext(db.Statement.Context))))
	fmt.Printf("%-23s | %5d | %s\n", "Debug()", getCloneValue(db.Debug()), cloneToImmutable(getCloneValue(db.Debug())))
	fmt.Printf("%-23s | %5d | %s\n", "Begin()", getCloneValue(db.Begin()), cloneToImmutable(getCloneValue(db.Begin())))

	// Chain methods (expected: clone=1)
	fmt.Printf("%-23s | %5d | %s\n", "Model()", getCloneValue(db.Model(&User{})), cloneToImmutable(getCloneValue(db.Model(&User{}))))
	fmt.Printf("%-23s | %5d | %s\n", "Table()", getCloneValue(db.Table("users")), cloneToImmutable(getCloneValue(db.Table("users"))))
	fmt.Printf("%-23s | %5d | %s\n", "Where()", getCloneValue(db.Where("x = ?", 1)), cloneToImmutable(getCloneValue(db.Where("x = ?", 1))))
	fmt.Printf("%-23s | %5d | %s\n", "Or()", getCloneValue(db.Or("x = ?", 1)), cloneToImmutable(getCloneValue(db.Or("x = ?", 1))))
	fmt.Printf("%-23s | %5d | %s\n", "Not()", getCloneValue(db.Not("x = ?", 1)), cloneToImmutable(getCloneValue(db.Not("x = ?", 1))))
	fmt.Printf("%-23s | %5d | %s\n", "Select()", getCloneValue(db.Select("id")), cloneToImmutable(getCloneValue(db.Select("id"))))
	fmt.Printf("%-23s | %5d | %s\n", "Omit()", getCloneValue(db.Omit("id")), cloneToImmutable(getCloneValue(db.Omit("id"))))
	fmt.Printf("%-23s | %5d | %s\n", "Order()", getCloneValue(db.Order("id")), cloneToImmutable(getCloneValue(db.Order("id"))))
	fmt.Printf("%-23s | %5d | %s\n", "Limit()", getCloneValue(db.Limit(10)), cloneToImmutable(getCloneValue(db.Limit(10))))
	fmt.Printf("%-23s | %5d | %s\n", "Offset()", getCloneValue(db.Offset(10)), cloneToImmutable(getCloneValue(db.Offset(10))))
	fmt.Printf("%-23s | %5d | %s\n", "Group()", getCloneValue(db.Group("id")), cloneToImmutable(getCloneValue(db.Group("id"))))
	fmt.Printf("%-23s | %5d | %s\n", "Having()", getCloneValue(db.Having("count(*) > ?", 1)), cloneToImmutable(getCloneValue(db.Having("count(*) > ?", 1))))
	fmt.Printf("%-23s | %5d | %s\n", "Joins()", getCloneValue(db.Joins("Profile")), cloneToImmutable(getCloneValue(db.Joins("Profile"))))
	fmt.Printf("%-23s | %5d | %s\n", "Distinct()", getCloneValue(db.Distinct()), cloneToImmutable(getCloneValue(db.Distinct())))
	fmt.Printf("%-23s | %5d | %s\n", "Unscoped()", getCloneValue(db.Unscoped()), cloneToImmutable(getCloneValue(db.Unscoped())))
	fmt.Printf("%-23s | %5d | %s\n", "Preload()", getCloneValue(db.Preload("Profile")), cloneToImmutable(getCloneValue(db.Preload("Profile"))))
	fmt.Printf("%-23s | %5d | %s\n", "Clauses()", getCloneValue(db.Clauses(clause.Locking{Strength: "UPDATE"})), cloneToImmutable(getCloneValue(db.Clauses(clause.Locking{Strength: "UPDATE"}))))
	fmt.Printf("%-23s | %5d | %s\n", "Attrs()", getCloneValue(db.Attrs("name", "test")), cloneToImmutable(getCloneValue(db.Attrs("name", "test"))))
	fmt.Printf("%-23s | %5d | %s\n", "Assign()", getCloneValue(db.Assign("name", "test")), cloneToImmutable(getCloneValue(db.Assign("name", "test"))))
	fmt.Printf("%-23s | %5d | %s\n", "Raw()", getCloneValue(db.Raw("SELECT 1")), cloneToImmutable(getCloneValue(db.Raw("SELECT 1"))))

	fmt.Println("\n=== Clone Value Meaning ===")
	fmt.Println("0 = No clone (dangerous! modifies receiver)")
	fmt.Println("1 = Clone Statement only (safe for immutable-return)")
	fmt.Println("2 = Full clone (Session-style, fully isolated)")

	// Callback argument clone values (CRITICAL!)
	fmt.Println("\n=== CALLBACK ARGUMENT CLONE VALUES ===")
	fmt.Println("(These determine if callback mutations leak)")
	fmt.Println("Method                  | Callback Clone | Safe?")
	fmt.Println("------------------------|----------------|------")

	// Scopes callback - needs query execution to trigger
	db7, mock7, _, _ := setupDBWithMock()
	var scopesClone int64 = -1
	mock7.ExpectQuery(".*").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	var users7 []User
	db7.Model(&User{}).Scopes(func(tx *gorm.DB) *gorm.DB {
		scopesClone = getCloneValue(tx)
		return tx
	}).Find(&users7)
	fmt.Printf("%-23s | %14d | %s\n", "Scopes()", scopesClone, cloneToSafe(scopesClone))

	// Transaction callback
	db8, mock8, _, _ := setupDBWithMock()
	var txClone int64 = -1
	mock8.ExpectBegin()
	mock8.ExpectCommit()
	_ = db8.Transaction(func(tx *gorm.DB) error {
		txClone = getCloneValue(tx)
		return nil
	})
	fmt.Printf("%-23s | %14d | %s\n", "Transaction()", txClone, cloneToSafe(txClone))

	// Connection callback (may not exist in older versions)
	var connClone int64 = -1
	db9, _, _, _ := setupDBWithMock()
	connMethod := reflect.ValueOf(db9).MethodByName("Connection")
	if connMethod.IsValid() {
		callback := func(tx *gorm.DB) error {
			connClone = getCloneValue(tx)
			return nil
		}
		connMethod.Call([]reflect.Value{reflect.ValueOf(callback)})
		fmt.Printf("%-23s | %14d | %s\n", "Connection()", connClone, cloneToSafe(connClone))
	} else {
		fmt.Printf("%-23s | %14s | %s\n", "Connection()", "N/A", "(method not found)")
	}

	// Note: Preload callback requires actual association query
	fmt.Printf("%-23s | %14s | %s\n", "Preload(callback)", "needs assoc", "(test separately)")

	fmt.Println("\n=== Pollution Test (Statement changes) ===")
	db2, _ := setupDB()
	base := db2.Model(&User{})
	fmt.Printf("Before Clauses: %v\n", base.Statement.Clauses)
	base.Where("marker = ?", 1) // discard
	fmt.Printf("After Clauses: %v\n", base.Statement.Clauses)
	_, hasWhere := base.Statement.Clauses["WHERE"]
	fmt.Printf("Where pollutes receiver: %v\n", hasWhere)

	// Accumulation test
	fmt.Println("\n=== Accumulation Test ===")
	db3, _ := setupDB()
	base3 := db3.Model(&User{})
	base3.Where("first = ?", 1)
	base3.Where("second = ?", 2)
	json3 := statementDumper.DumpJSONStr(base3.Statement)
	hasFirst := strings.Contains(json3, "first")
	hasSecond := strings.Contains(json3, "second")
	fmt.Printf("Where: first=%v second=%v → %s\n", hasFirst, hasSecond, func() string {
		if hasFirst && hasSecond {
			return "ACCUMULATE ☠️"
		}
		return "OVERWRITE ⚠️"
	}())

	fmt.Println("\n=== CRITICAL FINDING ===")
	fmt.Println("If Preload callback's clone=0, the callback's mutations")
	fmt.Println("will accumulate on repeated queries! (GitHub #7662)")
}

func cloneToSafe(clone int64) string {
	if clone >= 1 {
		return "✅ SAFE"
	}
	if clone == 0 {
		return "☠️ DANGEROUS (mutations leak!)"
	}
	return "? UNKNOWN"
}

func cloneToImmutable(clone int64) string {
	if clone >= 1 {
		return "✅ IMMUTABLE"
	}
	if clone == 0 {
		return "❌ MUTABLE"
	}
	return "? UNKNOWN"
}
