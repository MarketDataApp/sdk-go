// Package parameters defines structures and functions for handling request parameters across various API endpoints.
// It includes types for universal parameters, specific request types like stock quotes, options, and user inputs,
// and utilities for parsing and setting these parameters in API requests. The package leverages reflection
// for dynamic parameter parsing and validation, ensuring that API requests are constructed correctly.
package parameters

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/MarketDataApp/sdk-go/helpers/types" // Ensure you import the package where IsZeroValue is defined
)

type MarketDataParam interface {
	SetParams(*resty.Request) error
}

func ParseAndSetParams(params MarketDataParam, request *resty.Request) error {
    if params == nil {
        return errors.New("params cannot be nil")
    }
    kind := reflect.TypeOf(params).Kind()
    if kind != reflect.Struct && kind != reflect.Ptr {
        return fmt.Errorf("params must be a struct or a pointer to a struct, got %v", kind)
    }
    v := reflect.ValueOf(params)

    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }

    if v.Kind() != reflect.Struct {
        return fmt.Errorf("dereferenced value of params must be a struct, got %v", v.Kind())
    }

    t := v.Type()

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        value := v.Field(i)
        tag := field.Tag

        // Skip setting the parameter if it's not required and has a zero value.
        if !strings.Contains(tag.Get("path"), "required") && !strings.Contains(tag.Get("query"), "required") && types.IsZeroValue(value.Interface()) {
            continue
        }

        // Prepare the value for setting, handling pointers correctly.
        var valueInterface interface{}
        if value.Kind() == reflect.Ptr {
            if value.IsNil() {
                // Skip nil pointers unless they are required.
                continue
            }
            valueInterface = reflect.Indirect(value).Interface()
        } else {
            if types.IsZeroValue(value.Interface()) {
                // Skip zero values unless they are required.
                continue
            }
            valueInterface = value.Interface()
        }

        // Set the field to the appropriate part of the request.
        if pathTag := tag.Get("path"); pathTag != "" {
            request.SetPathParam(pathTag, fmt.Sprint(valueInterface))
        } else if queryTag := tag.Get("query"); queryTag != "" {
            request.SetQueryParam(queryTag, fmt.Sprint(valueInterface))
        }
    }

    return nil
}