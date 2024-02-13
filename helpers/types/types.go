// Package types provides utility functions for checking the type and value of variables.
// It includes methods for determining if a value is a "zero" value for its type,
// such as empty strings, nil pointers, zero numeric values, and more. This package
// is essential for validation and conditional logic based on variable states across
// the application. It leverages the reflect package to inspect types at runtime.
package types

import (
	"reflect"
	"time"
)

// IsZeroValue checks if the provided interface{} 'i' holds a value that is considered "zero" or "empty" for its type.
// For pointers, it also checks if the pointed-to value is a zero value, except for pointers to integers or booleans.
//
// This includes:
//   - nil pointers
//   - empty strings
//   - false for booleans
//   - zero for numeric types (integers, unsigned integers, floats)
//   - nil for slices, maps, interfaces, and channels
//   - time.Time structs representing the zero time
//   - structs where all fields are "zero" values
//
// # Parameters
//
//   - i: The interface{} to check for a "zero" or "empty" value.
//
// # Returns
//
//   - A bool indicating if 'i' is a "zero" or "empty" value.
func IsZeroValue(i interface{}) bool {
	// Check if the interface is nil, which is considered a zero value.
	if i == nil {
		return true
	}

	// Get the reflection Value of the interface.
	v := reflect.ValueOf(i)

	// Switch on the kind of the value to handle different types.
	switch v.Kind() {
	case reflect.String:
		// For strings, check if the length is zero.
		return v.Len() == 0
	case reflect.Bool:
		// For booleans, check if the value is false.
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// For integers, check if the value is zero.
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		// For unsigned integers, check if the value is zero.
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		// For floats, check if the value is zero.
		return v.Float() == 0
	case reflect.Ptr:
		// For pointers, first check if the pointer itself is nil.
		if v.IsNil() {
			return true
		}
		// Get the element the pointer points to.
		elem := v.Elem()
		// Special cases: if the pointer points to an int or bool, do not consider it a zero value.
		// First, check for all integer types.
		if elem.Kind() == reflect.Int || elem.Kind() == reflect.Int8 || elem.Kind() == reflect.Int16 || elem.Kind() == reflect.Int32 || elem.Kind() == reflect.Int64 {
			return false
		}
		// Subsequent check for bool.
		if elem.Kind() == reflect.Bool {
			return false
		}
		// Otherwise, check if the pointed-to value is a zero value.
		return IsZeroValue(elem.Interface())
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan:
		// For slices, maps, interfaces, and channels, check if they are nil.
		if v.IsNil() {
			return true
		}
		// Additionally, for slices and maps, check if they are non-nil but empty.
		return v.Len() == 0
	case reflect.Struct:
		// Check if the struct has an IsZero method.
		if method := v.MethodByName("IsZero"); method.IsValid() {
			// Ensure the method has the correct signature: func() bool
			if method.Type().NumIn() == 0 && method.Type().NumOut() == 1 && method.Type().Out(0).Kind() == reflect.Bool {
				// Call the IsZero method and return its result.
				result := method.Call(nil)
				if len(result) == 1 && result[0].Kind() == reflect.Bool {
					return result[0].Bool()
				}
			}
		}
		// Special case for time.Time: check if it represents the zero time.
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		// For other structs, check each field. If any field is not a zero value, the struct is not a zero value.
		for i := 0; i < v.NumField(); i++ {
			if !IsZeroValue(v.Field(i).Interface()) {
				return false
			}
		}
		// If all fields are zero values, the struct is a zero value.
		return true
	default:
		// For any other type, consider it not a zero value.
		// This includes types like channels, which are not handled explicitly.
		return false
	}
}

// IsAlpha checks if the input string consists only of alphabetic characters.
//
// # Parameters
//
//   - s: The string to be checked.
//
// # Returns
//
//   - A boolean value indicating whether the string is alphabetic (true) or not (false).
func IsAlpha(s string) bool {
	for _, r := range s {
		if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z') {
			return false
		}
	}
	return true
}
