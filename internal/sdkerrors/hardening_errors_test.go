package sdkerrors

import (
	"errors"
	"strings"
	"testing"
)

func TestResponseTooLargeError(t *testing.T) {
	cause := errors.New("boom")
	e := &ResponseTooLargeError{Limit: 1024, Cause: cause}
	if !strings.Contains(e.Error(), "1024") {
		t.Errorf("Error() = %q, want it to mention the limit", e.Error())
	}
	if e.Retryable() {
		t.Error("Retryable() = true, want false")
	}
	if !errors.Is(e, ErrResponseTooLarge) {
		t.Error("errors.Is(e, ErrResponseTooLarge) = false")
	}
	if !errors.Is(e.Unwrap(), cause) {
		t.Error("Unwrap() did not return the cause")
	}
	var target *ResponseTooLargeError
	if !errors.As(error(e), &target) {
		t.Error("errors.As failed")
	}
}

func TestInsecureTokenError(t *testing.T) {
	e := &InsecureTokenError{Scheme: "http", Host: "evil.example.com"}
	msg := e.Error()
	if !strings.Contains(msg, "http") || !strings.Contains(msg, "evil.example.com") {
		t.Errorf("Error() = %q, want scheme and host", msg)
	}
	if e.Retryable() {
		t.Error("Retryable() = true, want false")
	}
	if e.Unwrap() != nil {
		t.Error("Unwrap() != nil")
	}
	if !errors.Is(e, ErrInsecureToken) {
		t.Error("errors.Is(e, ErrInsecureToken) = false")
	}
	// Distinct sentinels must not cross-match.
	if errors.Is(e, ErrResponseTooLarge) {
		t.Error("InsecureTokenError incorrectly matched ErrResponseTooLarge")
	}
}
