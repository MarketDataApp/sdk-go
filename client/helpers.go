package client

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
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

// DecodeDate decodes a date from an interface{} type.
// It supports time.Time and string types and returns the date in "YYYY-MM-DD" format and any error encountered.
func DecodeDate(date interface{}) (string, error) {
	switch v := date.(type) {
	case time.Time:
		return v.Format("2006-01-02"), nil
	case string:
		_, err := time.Parse("2006-01-02", v)
		if err != nil {
			return "", err
		}
		return v, nil
	default:
		return "", errors.New("date must be a time.Time object or a YYYY-MM-DD string")
	}
}

// ParseAndSetParams takes a struct and a Resty request, parses the struct into path and query parameters, and sets them to the request.
// It returns an error if a required parameter has a zero value.
func parseAndSetParams(params MarketDataParam, request *resty.Request) error {
	if reflect.TypeOf(params).Kind() != reflect.Struct {
		return errors.New("params must be a struct")
	}
	v := reflect.ValueOf(params)

	// Check if the params is a pointer and dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	// Iterate over the fields of the struct.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get the field tag value.
		tag := field.Tag

		// Check if the field is required and has a zero value.
		if (strings.Contains(tag.Get("path"), "required") || strings.Contains(tag.Get("query"), "required")) && value.IsZero() {
			return fmt.Errorf("required parameter %s has a zero value", field.Name)
		}

		// Set the field to the appropriate part of the request if it is not a zero value.
		if !value.IsZero() {
			if pathTag := tag.Get("path"); pathTag != "" {
				request.SetPathParam(pathTag, fmt.Sprint(value.Interface()))
			} else if queryTag := tag.Get("query"); queryTag != "" {
				request.SetQueryParam(queryTag, fmt.Sprint(value.Interface()))
			}
		}
	}

	return nil
}
