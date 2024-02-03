package types

import (
	"testing"
	"time"
)

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"nil", nil, true},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"false boolean", false, true},
		{"true boolean", true, false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero unsigned int", uint(0), true},
		{"non-zero unsigned int", uint(9), false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer to zero int", new(int), false},
		{"non-nil pointer to non-zero int", func() *int { var i = 42; return &i }(), false},
		{"nil pointer to bool", (*bool)(nil), true},
		{"non-nil pointer to false bool", new(bool), false},
		{"non-nil pointer to true bool", func() *bool { b := true; return &b }(), false},
		{"nil pointer to int", (*int)(nil), true},
		{"non-nil pointer to zero int", new(int), false},
		{"non-nil pointer to non-zero int", func() *int { var i = 42; return &i }(), false},
		{"nil slice", []int(nil), true},
		{"non-nil empty slice", []int{}, true},
		{"non-nil non-empty slice", []int{1}, false},
		{"nil map", map[string]int(nil), true},
		{"non-nil empty map", map[string]int{}, true},
		{"non-nil non-empty map", map[string]int{"a": 1}, false},
		{"zero time", time.Time{}, true},
		{"non-zero time", time.Now(), false},
		{"struct with zero values", struct {
			A int
			B string
		}{}, true},
		{"struct with non-zero values", struct {
			A int
			B string
		}{A: 1, B: "a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsZeroValue(tt.input)
			if result != tt.expected {
				t.Errorf("IsZeroValue(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsAlpha(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"all letters", "TestString", true},
		{"contains digits", "Test123", false},
		{"empty string", "", true},
		{"special characters", "Test!", false},
		{"spaces", "Test String", false},
		{"only letters lowercase", "teststring", true},
		{"only letters uppercase", "TESTSTRING", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAlpha(tt.input)
			if result != tt.expected {
				t.Errorf("IsAlpha(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}