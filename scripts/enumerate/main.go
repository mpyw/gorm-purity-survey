// Package main enumerates all public methods of *gorm.DB and recursively derived types.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// MethodInfo holds information about a method.
type MethodInfo struct {
	Name        string   `json:"name"`
	NumIn       int      `json:"num_in"`
	NumOut      int      `json:"num_out"`
	InTypes     []string `json:"in_types"`
	OutTypes    []string `json:"out_types"`
	ReturnsDB   bool     `json:"returns_db"`
	TakesDB     bool     `json:"takes_db"`
	TakesDBFunc bool     `json:"takes_db_func"` // Takes func that receives/returns *gorm.DB
	Variadic    bool     `json:"variadic"`
	Signature   string   `json:"signature"`
}

// TypeMethods holds methods for a specific type.
type TypeMethods struct {
	TypeName     string       `json:"type_name"`
	MethodCount  int          `json:"method_count"`
	Methods      []MethodInfo `json:"methods"`
	DerivedTypes []string     `json:"derived_types,omitempty"` // Types returned by methods
}

// EnumerationResult holds the complete enumeration result.
type EnumerationResult struct {
	GormVersion    string                 `json:"gorm_version"`
	Types          map[string]TypeMethods `json:"types"`
	PollutionPaths []string               `json:"pollution_paths"` // Ways *gorm.DB can be polluted
}

// TypeEnumerator handles recursive type enumeration.
type TypeEnumerator struct {
	visited map[string]bool
	result  map[string]TypeMethods
	toVisit []reflect.Type
}

func main() {
	enumerator := &TypeEnumerator{
		visited: make(map[string]bool),
		result:  make(map[string]TypeMethods),
	}

	// Start with root types
	rootTypes := []interface{}{
		&gorm.DB{},
		&gorm.Association{},
		&gorm.Statement{},
		gorm.Session{},
		gorm.Config{},
		&schema.Schema{},
		clause.Clause{},
	}

	// Add interface types (Generics API) - available in GORM v1.25+
	// These interfaces hold internal *gorm.DB and need investigation
	interfaceTypes := getGenericsAPITypes()

	for _, root := range rootTypes {
		t := reflect.TypeOf(root)
		enumerator.enumerateRecursive(t)
	}

	// Enumerate interface types (Generics API)
	for _, t := range interfaceTypes {
		enumerator.enumerateRecursive(t)
	}

	// Process queued types
	for len(enumerator.toVisit) > 0 {
		t := enumerator.toVisit[0]
		enumerator.toVisit = enumerator.toVisit[1:]
		enumerator.enumerateRecursive(t)
	}

	// Analyze pollution paths
	pollutionPaths := enumerator.findPollutionPaths()

	// Get GORM version from environment or file
	gormVersion := os.Getenv("GORM_VERSION")
	if gormVersion == "" {
		if data, err := os.ReadFile("/tmp/gorm_version.txt"); err == nil {
			gormVersion = strings.TrimSpace(string(data))
		} else {
			gormVersion = "unknown"
		}
	}

	output := EnumerationResult{
		GormVersion:    gormVersion,
		Types:          enumerator.result,
		PollutionPaths: pollutionPaths,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func (e *TypeEnumerator) enumerateRecursive(t reflect.Type) {
	typeName := t.String()

	// Skip if already visited
	if e.visited[typeName] {
		return
	}
	e.visited[typeName] = true

	// Skip primitive types and common stdlib types
	if shouldSkipType(typeName) {
		return
	}

	// Only enumerate types with methods
	numMethods := t.NumMethod()
	if numMethods == 0 {
		return
	}

	methods := make([]MethodInfo, 0)
	derivedTypes := make(map[string]bool)

	for i := 0; i < numMethods; i++ {
		m := t.Method(i)
		if !m.IsExported() {
			continue
		}

		mt := m.Type
		info := MethodInfo{
			Name:     m.Name,
			NumIn:    mt.NumIn() - 1,
			NumOut:   mt.NumOut(),
			Variadic: mt.IsVariadic(),
		}

		// Collect input types
		for j := 1; j < mt.NumIn(); j++ {
			inType := mt.In(j)
			typeName := formatType(inType, mt.IsVariadic() && j == mt.NumIn()-1)
			info.InTypes = append(info.InTypes, typeName)

			if isGormDB(inType) {
				info.TakesDB = true
			}
			if isGormDBFunc(inType) {
				info.TakesDBFunc = true
			}
		}

		// Collect output types and queue them for recursive enumeration
		for j := 0; j < mt.NumOut(); j++ {
			outType := mt.Out(j)
			outTypeName := outType.String()
			info.OutTypes = append(info.OutTypes, outTypeName)

			if isGormDB(outType) {
				info.ReturnsDB = true
			}

			// Queue non-primitive return types for recursive enumeration
			if shouldEnumerateType(outType) && !e.visited[outTypeName] {
				derivedTypes[outTypeName] = true
				e.toVisit = append(e.toVisit, outType)
			}
		}

		info.Signature = buildSignature(info)
		methods = append(methods, info)
	}

	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Name < methods[j].Name
	})

	derivedList := make([]string, 0, len(derivedTypes))
	for dt := range derivedTypes {
		derivedList = append(derivedList, dt)
	}
	sort.Strings(derivedList)

	e.result[typeName] = TypeMethods{
		TypeName:     typeName,
		MethodCount:  len(methods),
		Methods:      methods,
		DerivedTypes: derivedList,
	}
}

func (e *TypeEnumerator) findPollutionPaths() []string {
	var paths []string

	for typeName, tm := range e.result {
		for _, m := range tm.Methods {
			// Methods that take *gorm.DB directly
			if m.TakesDB {
				paths = append(paths, fmt.Sprintf("%s.%s takes *gorm.DB directly", typeName, m.Name))
			}

			// Methods that take func(*gorm.DB)
			if m.TakesDBFunc {
				paths = append(paths, fmt.Sprintf("%s.%s takes func with *gorm.DB", typeName, m.Name))
			}

			// Methods that return *gorm.DB (potential chain point)
			if m.ReturnsDB {
				paths = append(paths, fmt.Sprintf("%s.%s returns *gorm.DB (chain point)", typeName, m.Name))
			}
		}
	}

	sort.Strings(paths)
	return paths
}

func shouldSkipType(typeName string) bool {
	// Skip primitive types
	primitives := []string{
		"error", "string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64",
		"uintptr", "byte", "rune", "complex64", "complex128",
	}
	for _, p := range primitives {
		if typeName == p {
			return true
		}
	}

	// Skip stdlib types that aren't relevant
	skipPrefixes := []string{
		"*sql.", "sql.", "context.", "time.", "sync.", "io.",
		"*bytes.", "bytes.", "[]byte", "[]uint8",
		"map[string]", "[]string", "[]int",
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(typeName, prefix) {
			return true
		}
	}

	// Skip interface{} and any
	if typeName == "interface {}" || typeName == "any" {
		return true
	}

	return false
}

func shouldEnumerateType(t reflect.Type) bool {
	typeName := t.String()

	// Skip if it's a type we want to skip
	if shouldSkipType(typeName) {
		return false
	}

	// Must have methods
	if t.NumMethod() == 0 {
		return false
	}

	// Should be a gorm-related type
	gormRelated := []string{"gorm.", "clause.", "schema."}
	for _, prefix := range gormRelated {
		if strings.Contains(typeName, prefix) {
			return true
		}
	}

	return false
}

func isGormDB(t reflect.Type) bool {
	return t.String() == "*gorm.DB"
}

func isGormDBFunc(t reflect.Type) bool {
	if t.Kind() != reflect.Func {
		return false
	}

	// Check if any parameter or return value is *gorm.DB
	for i := 0; i < t.NumIn(); i++ {
		if isGormDB(t.In(i)) {
			return true
		}
	}
	for i := 0; i < t.NumOut(); i++ {
		if isGormDB(t.Out(i)) {
			return true
		}
	}
	return false
}

func formatType(t reflect.Type, isVariadicParam bool) string {
	if isVariadicParam {
		return "..." + t.Elem().String()
	}
	return t.String()
}

func buildSignature(info MethodInfo) string {
	sig := info.Name + "("
	for i, in := range info.InTypes {
		if i > 0 {
			sig += ", "
		}
		sig += in
	}
	sig += ")"

	if len(info.OutTypes) > 0 {
		if len(info.OutTypes) == 1 {
			sig += " " + info.OutTypes[0]
		} else {
			sig += " ("
			for i, out := range info.OutTypes {
				if i > 0 {
					sig += ", "
				}
				sig += out
			}
			sig += ")"
		}
	}

	return sig
}
