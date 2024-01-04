package client

import (
	"errors"
	"fmt"
	"net/http"
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

// ParseAndSetParams takes a struct and a Resty request, parses the struct into path and query parameters, and sets them to the request.
// It returns an error if a required parameter has a zero value.
func parseAndSetParams(params MarketDataParam, request *resty.Request) error {
	if params == nil {
		return errors.New("params cannot be nil")
	}
	kind := reflect.TypeOf(params).Kind()
	if kind != reflect.Struct && kind != reflect.Ptr {
		return errors.New("params must be a struct or a pointer to a struct")
	}
	v := reflect.ValueOf(params)

	// Check if the params is a pointer and dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if the dereferenced value is a struct
	if v.Kind() != reflect.Struct {
		return errors.New("params must be a struct or a pointer to a struct")
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
			var valueInterface interface{}
			if value.Kind() == reflect.Ptr && !value.IsNil() {
				valueInterface = reflect.Indirect(value).Interface()
			} else {
				valueInterface = value.Interface()
			}
			if pathTag := tag.Get("path"); pathTag != "" {
				request.SetPathParam(pathTag, fmt.Sprint(valueInterface))
			} else if queryTag := tag.Get("query"); queryTag != "" {
				request.SetQueryParam(queryTag, fmt.Sprint(valueInterface))
			}
		}
	}

	return nil
}

func redactAuthorizationHeader(headers http.Header) http.Header {
	// Copy the headers so we don't modify the original
	copiedHeaders := make(http.Header)
	for k, v := range headers {
		copiedHeaders[k] = v
	}

	// Redact the Authorization header if it exists
	if _, ok := copiedHeaders["Authorization"]; ok {
		token := copiedHeaders.Get("Authorization")
		redactedToken := "Bearer " + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
		copiedHeaders.Set("Authorization", redactedToken)
	}

	return copiedHeaders
}
