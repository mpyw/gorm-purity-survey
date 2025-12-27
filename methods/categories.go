// Package methods defines method categories for the GORM purity survey.
package methods

// Category represents a method category for survey prioritization.
type Category string

const (
	// CategoryImmutableReturn - Known immutable-returning methods (already in gormreuse)
	CategoryImmutableReturn Category = "immutable-return"

	// CategoryChain - Chain methods that modify query state
	CategoryChain Category = "chain"

	// CategoryFinisher - Methods that execute queries
	CategoryFinisher Category = "finisher"

	// CategoryTransaction - Transaction-related methods
	CategoryTransaction Category = "transaction"

	// CategoryCallback - Methods taking func(*gorm.DB) callbacks
	CategoryCallback Category = "callback"

	// CategoryUtility - Utility methods (not query-related)
	CategoryUtility Category = "utility"

	// CategoryInterfaceArg - Methods with interface{} args needing deep investigation
	CategoryInterfaceArg Category = "interface-arg"
)

// Priority represents survey priority.
type Priority int

const (
	PriorityHigh   Priority = 1 // Likely to have version-dependent behavior
	PriorityMedium Priority = 2 // May have edge cases
	PriorityLow    Priority = 3 // Unlikely to cause issues
)

// MethodInfo holds categorization info for a method.
type MethodInfo struct {
	Name            string
	Category        Category
	Priority        Priority
	ReturnsDB       bool // Method returns *gorm.DB
	TakesDB         bool // Method takes *gorm.DB as argument
	TakesDBCallback bool // Method takes func(*gorm.DB) callback
	HasInterfaceArg bool
	Notes           string
}

// Methods is the categorized list of all *gorm.DB methods.
var Methods = []MethodInfo{
	// === Known Immutable-Return (gormreuse built-in) ===
	{Name: "Session", Category: CategoryImmutableReturn, Priority: PriorityLow, ReturnsDB: true, Notes: "Creates new Statement"},
	{Name: "WithContext", Category: CategoryImmutableReturn, Priority: PriorityLow, ReturnsDB: true, Notes: "Calls Session internally"},
	{Name: "Debug", Category: CategoryImmutableReturn, Priority: PriorityLow, ReturnsDB: true, Notes: "Calls Session internally"},
	{Name: "Begin", Category: CategoryImmutableReturn, Priority: PriorityLow, ReturnsDB: true, Notes: "Creates new transaction"},

	// === Chain Methods (HIGH PRIORITY - likely pollute) ===
	{Name: "Where", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Core query builder"},
	{Name: "Or", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Core query builder"},
	{Name: "Not", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Core query builder"},
	{Name: "Select", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "interface{} patterns may vary"},
	{Name: "Order", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "interface{} patterns may vary"},
	{Name: "Group", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, Notes: "String only"},
	{Name: "Having", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Similar to Where"},
	{Name: "Limit", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, Notes: "int only"},
	{Name: "Offset", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, Notes: "int only"},
	{Name: "Joins", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "String + args"},
	{Name: "InnerJoins", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "String + args"},
	{Name: "Preload", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "May have func callback"},
	{Name: "Clauses", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, Notes: "Direct clause manipulation"},
	{Name: "Table", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Table name + args"},
	{Name: "Model", Category: CategoryChain, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Sets model type"},
	{Name: "Distinct", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Distinct columns"},
	{Name: "Omit", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, Notes: "String columns only"},
	{Name: "Unscoped", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, Notes: "Disables soft delete"},
	{Name: "Raw", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Raw SQL"},
	{Name: "Assign", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "FirstOrCreate attrs"},
	{Name: "Attrs", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "FirstOrCreate attrs"},
	{Name: "Set", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Set key-value"},
	{Name: "InstanceSet", Category: CategoryChain, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Instance key-value"},
	{Name: "MapColumns", Category: CategoryChain, Priority: PriorityLow, ReturnsDB: true, Notes: "Column mapping"},

	// === Finishers (MEDIUM PRIORITY - need pollution check) ===
	{Name: "Find", Category: CategoryFinisher, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "Primary finisher"},
	{Name: "First", Category: CategoryFinisher, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "With optional conds"},
	{Name: "Take", Category: CategoryFinisher, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "With optional conds"},
	{Name: "Last", Category: CategoryFinisher, Priority: PriorityHigh, ReturnsDB: true, HasInterfaceArg: true, Notes: "With optional conds"},
	{Name: "FirstOrInit", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "May modify state"},
	{Name: "FirstOrCreate", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "May modify state"},
	{Name: "Create", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Insert"},
	{Name: "CreateInBatches", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Batch insert"},
	{Name: "Save", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Upsert"},
	{Name: "Update", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Single column"},
	{Name: "Updates", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Multiple columns"},
	{Name: "UpdateColumn", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Skip hooks"},
	{Name: "UpdateColumns", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Skip hooks"},
	{Name: "Delete", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Soft/hard delete"},
	{Name: "Count", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, Notes: "Aggregate"},
	{Name: "Pluck", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Single column"},
	{Name: "Scan", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Raw scan"},
	{Name: "Row", Category: CategoryFinisher, Priority: PriorityLow, ReturnsDB: false, Notes: "Returns *sql.Row"},
	{Name: "Rows", Category: CategoryFinisher, Priority: PriorityLow, ReturnsDB: false, Notes: "Returns *sql.Rows"},
	{Name: "Exec", Category: CategoryFinisher, Priority: PriorityMedium, ReturnsDB: true, HasInterfaceArg: true, Notes: "Raw exec"},

	// === Transaction Methods ===
	{Name: "Commit", Category: CategoryTransaction, Priority: PriorityMedium, ReturnsDB: true, Notes: "End transaction"},
	{Name: "Rollback", Category: CategoryTransaction, Priority: PriorityMedium, ReturnsDB: true, Notes: "End transaction"},
	{Name: "SavePoint", Category: CategoryTransaction, Priority: PriorityLow, ReturnsDB: true, Notes: "Savepoint"},
	{Name: "RollbackTo", Category: CategoryTransaction, Priority: PriorityLow, ReturnsDB: true, Notes: "Rollback to savepoint"},

	// === Callback Methods (need isolation check) ===
	{Name: "Scopes", Category: CategoryCallback, Priority: PriorityHigh, ReturnsDB: true, TakesDBCallback: true, Notes: "Applies scope funcs - isolation?"},
	{Name: "Transaction", Category: CategoryCallback, Priority: PriorityMedium, ReturnsDB: false, TakesDBCallback: true, Notes: "Callback isolation?"},
	{Name: "Connection", Category: CategoryCallback, Priority: PriorityMedium, ReturnsDB: false, TakesDBCallback: true, Notes: "Callback isolation?"},
	{Name: "FindInBatches", Category: CategoryCallback, Priority: PriorityMedium, ReturnsDB: true, TakesDBCallback: true, Notes: "Callback isolation?"},
	{Name: "ToSQL", Category: CategoryCallback, Priority: PriorityLow, ReturnsDB: false, TakesDBCallback: true, Notes: "SQL generation"},

	// === Methods that RECEIVE *gorm.DB directly ===
	{Name: "Initialize", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, TakesDB: true, Notes: "Plugin init - receives *gorm.DB"},
	{Name: "AfterInitialize", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, TakesDB: true, Notes: "Plugin hook - receives *gorm.DB"},

	// === Utility Methods (LOW PRIORITY) ===
	{Name: "DB", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Get *sql.DB"},
	{Name: "Migrator", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Schema migration"},
	{Name: "AutoMigrate", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Auto migration"},
	{Name: "Callback", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Callback registry"},
	{Name: "Use", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Plugin"},
	{Name: "SetupJoinTable", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Join table setup"},
	{Name: "Association", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Association handler"},
	{Name: "Get", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Get setting"},
	{Name: "InstanceGet", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Get instance setting"},
	{Name: "AddError", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Error handling"},
	{Name: "ScanRows", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Scan sql.Rows"},
	{Name: "Explain", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Query explain"},

	// === Dialect/Plugin Interface Methods (skip) ===
	{Name: "Name", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Dialect name"},
	{Name: "Apply", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Config apply"},
	{Name: "BindVarTo", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Dialect"},
	{Name: "QuoteTo", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Dialect"},
	{Name: "DataTypeOf", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Dialect"},
	{Name: "DefaultValueOf", Category: CategoryUtility, Priority: PriorityLow, ReturnsDB: false, Notes: "Dialect"},
}

// CallbackMethods returns methods with func(*gorm.DB) callbacks.
func CallbackMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.TakesDBCallback {
			result = append(result, m)
		}
	}
	return result
}

// ReceivesDBMethods returns methods that take *gorm.DB as argument.
func ReceivesDBMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.TakesDB || m.TakesDBCallback {
			result = append(result, m)
		}
	}
	return result
}

// HighPriorityMethods returns methods that should be surveyed first.
func HighPriorityMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.Priority == PriorityHigh {
			result = append(result, m)
		}
	}
	return result
}

// InterfaceArgMethods returns methods with interface{} arguments.
func InterfaceArgMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.HasInterfaceArg {
			result = append(result, m)
		}
	}
	return result
}

// ChainMethods returns chain methods.
func ChainMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.Category == CategoryChain {
			result = append(result, m)
		}
	}
	return result
}

// FinisherMethods returns finisher methods.
func FinisherMethods() []MethodInfo {
	var result []MethodInfo
	for _, m := range Methods {
		if m.Category == CategoryFinisher {
			result = append(result, m)
		}
	}
	return result
}
