package http

import "testing"

func TestPathSegment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain symbol", "AAPL", "AAPL"},
		{"dotted symbol", "BRK.B", "BRK.B"},
		{"hyphenated symbol", "BRK-B", "BRK-B"},
		{"caret index", "^SPX", "%5ESPX"},
		{"slash injection", "AAPL/../../user", "AAPL%2F..%2F..%2Fuser"},
		{"encoded slash stays encoded", "AAPL%2Fuser", "AAPL%252Fuser"},
		{"space", "BRK A", "BRK%20A"},
		{"query smuggling", "AAPL?human=true", "AAPL%3Fhuman=true"},
		{"single dot segment", ".", "%2E"},
		{"double dot segment", "..", "%2E%2E"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathSegment(tt.in); got != tt.want {
				t.Errorf("PathSegment(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeMessage(t *testing.T) {
	// Control characters are collapsed to spaces
	got := sanitizeMessage("line1\nline2\r\x1b[31mred\x00")
	want := "line1 line2  [31mred "
	if got != want {
		t.Errorf("sanitizeMessage() = %q, want %q", got, want)
	}

	// Long messages are truncated
	long := make([]byte, 2000)
	for i := range long {
		long[i] = 'a'
	}
	got = sanitizeMessage(string(long))
	if len(got) > maxErrorMessageLen+len("…(truncated)") {
		t.Errorf("sanitizeMessage() length = %d, want <= %d", len(got), maxErrorMessageLen+len("…(truncated)"))
	}

	// Short clean messages pass through unchanged
	if got := sanitizeMessage("Rate limit exceeded"); got != "Rate limit exceeded" {
		t.Errorf("sanitizeMessage() = %q, want unchanged", got)
	}
}

func TestRedactQuery(t *testing.T) {
	got := redactQuery("https://api.example.com/v1/stocks/quotes/AAPL/?human=true&columns=bid")
	want := "https://api.example.com/v1/stocks/quotes/AAPL/?…"
	if got != want {
		t.Errorf("redactQuery() = %q, want %q", got, want)
	}

	noQuery := "https://api.example.com/v1/stocks/quotes/AAPL/"
	if got := redactQuery(noQuery); got != noQuery {
		t.Errorf("redactQuery() = %q, want unchanged", got)
	}
}
