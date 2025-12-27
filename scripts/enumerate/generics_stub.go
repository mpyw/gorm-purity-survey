//go:build !gorm_v125plus

package main

import "reflect"

// getGenericsAPITypes returns empty slice for older GORM versions without Generics API
func getGenericsAPITypes() []reflect.Type {
	return nil
}
