package parameters

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-resty/resty/v2"
)

type MarketDataParam interface {
	SetParams(*resty.Request) error
}

// ParseAndSetParams takes a struct and a Resty request, parses the struct into path and query parameters, and sets them to the request.
// It returns an error if a required parameter has a zero value.
func ParseAndSetParams(params MarketDataParam, request *resty.Request) error {
	if params == nil {
		return errors.New("params cannot be nil")
	}
	kind := reflect.TypeOf(params).Kind()
	if kind != reflect.Struct && kind != reflect.Ptr {
		return fmt.Errorf("params must be a struct or a pointer to a struct, got %v", kind)
	}
	v := reflect.ValueOf(params)

	// Check if the params is a pointer and dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Check if the dereferenced value is a struct
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dereferenced value of params must be a struct, got %v", v.Kind())
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
