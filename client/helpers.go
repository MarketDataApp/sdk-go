package client

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

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

// ParseParams takes a slice of structs and returns two maps: one for path parameters and one for query parameters.
// It returns an error if a required parameter has a zero value.
func ParseParams(paramsSlice []interface{}) (map[string]string, map[string]string, error) {
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)

	for _, params := range paramsSlice {
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
				return nil, nil, fmt.Errorf("required parameter %s has a zero value", field.Name)
			}

			// Add the field to the appropriate map if it is not a zero value.
			if !value.IsZero() {
				if pathTag := tag.Get("path"); pathTag != "" {
					pathParams[pathTag] = fmt.Sprint(value.Interface())
				} else if queryTag := tag.Get("query"); queryTag != "" {
					queryParams[queryTag] = fmt.Sprint(value.Interface())
				}
			}
		}
	}

	return pathParams, queryParams, nil
}

// ParseAndSetParams takes a struct and a Resty request, parses the struct into path and query parameters, and sets them to the request.
// It returns an error if a required parameter has a zero value.
func ParseAndSetParams(params MarketDataParam, request *resty.Request) error {
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
