//go:build gorm_v125plus

package main

import (
	"reflect"

	"gorm.io/gorm"
)

// getGenericsAPITypes returns the Generics API interface types (available in GORM v1.30+)
func getGenericsAPITypes() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf((*gorm.PreloadBuilder)(nil)).Elem(),
		reflect.TypeOf((*gorm.JoinBuilder)(nil)).Elem(),
	}
}
