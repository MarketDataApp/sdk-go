package helpers

import (
	"reflect"
	"time"
)

func IsZeroValue(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan:
		return v.IsNil()
	case reflect.Struct:
		if _, ok := v.Interface().(time.Time); ok {
			return v.Interface() == time.Time{}
		}
		zero := true
		for i := 0; i < v.NumField(); i++ {
			if !IsZeroValue(v.Field(i).Interface()) {
				zero = false
				break
			}
		}
		return zero
	default:
		// Add more types if needed
		return false
	}
}

func IsAlpha(s string) bool {
	for _, r := range s {
		if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z') {
			return false
		}
	}
	return true
}
