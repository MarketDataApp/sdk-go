package response

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
)

func TestIsJSON(t *testing.T) {
	r := &Response{
		Response: &http.Response{
			Header: http.Header{"Content-Type": []string{"application/json; charset=utf-8"}},
		},
	}
	if !r.IsJSON() {
		t.Error("IsJSON() should return true for application/json")
	}

	r.Header.Set("Content-Type", "text/html")
	if r.IsJSON() {
		t.Error("IsJSON() should return false for text/html")
	}
}

func TestIsCSV(t *testing.T) {
	r := &Response{
		Response: &http.Response{
			Header: http.Header{"Content-Type": []string{"text/csv"}},
		},
	}
	if !r.IsCSV() {
		t.Error("IsCSV() should return true for text/csv")
	}

	r.Header.Set("Content-Type", "application/json")
	if r.IsCSV() {
		t.Error("IsCSV() should return false for application/json")
	}
}

func TestIsHTML(t *testing.T) {
	r := &Response{
		Response: &http.Response{
			Header: http.Header{"Content-Type": []string{"text/html"}},
		},
	}
	if !r.IsHTML() {
		t.Error("IsHTML() should return true for text/html")
	}

	r.Header.Set("Content-Type", "application/json")
	if r.IsHTML() {
		t.Error("IsHTML() should return false for application/json")
	}
}

func TestSaveToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	r := &Response{
		Response: &http.Response{
			Header: http.Header{},
		},
		body: []byte(`{"test": true}`),
	}

	err := r.SaveToFile(path)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != `{"test": true}` {
		t.Errorf("file content = %q, want %q", string(data), `{"test": true}`)
	}
}

func TestSaveToFile_NilBody(t *testing.T) {
	r := &Response{
		Response: &http.Response{
			Header: http.Header{},
		},
	}

	err := r.SaveToFile("/tmp/test_output.json")
	if err == nil {
		t.Fatal("SaveToFile() should return error for nil body")
	}
}

func TestNew(t *testing.T) {
	httpResp := &internalhttp.Response{
		Raw: &http.Response{
			Header: http.Header{
				"X-Api-Ratelimit-Limit":     []string{"10000"},
				"X-Api-Ratelimit-Remaining": []string{"9999"},
				"X-Api-Ratelimit-Consumed":  []string{"1"},
				"X-Api-Ratelimit-Reset":     []string{"1704067200"},
			},
		},
		Headers: http.Header{
			"X-Api-Ratelimit-Limit":     []string{"10000"},
			"X-Api-Ratelimit-Remaining": []string{"9999"},
			"X-Api-Ratelimit-Consumed":  []string{"1"},
			"X-Api-Ratelimit-Reset":     []string{"1704067200"},
		},
		Body: []byte(`{"s":"ok"}`),
	}

	r := New(httpResp)
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.NoData {
		t.Error("NoData should be false")
	}
	if r.RateLimit.Limit != 10000 {
		t.Errorf("RateLimit.Limit = %d, want 10000", r.RateLimit.Limit)
	}
	if r.RateLimit.Remaining != 9999 {
		t.Errorf("RateLimit.Remaining = %d, want 9999", r.RateLimit.Remaining)
	}
	if r.RateLimit.Consumed != 1 {
		t.Errorf("RateLimit.Consumed = %d, want 1", r.RateLimit.Consumed)
	}
	if r.RateLimit.ResetAt.IsZero() {
		t.Error("RateLimit.ResetAt should not be zero")
	}
}

func TestNewCSV(t *testing.T) {
	httpResp := &internalhttp.Response{
		Raw: &http.Response{Header: http.Header{}, StatusCode: 200},
		Headers: http.Header{
			"X-Api-Ratelimit-Limit":     []string{"10000"},
			"X-Api-Ratelimit-Remaining": []string{"9999"},
		},
		Body: []byte("symbol,last\nAAPL,150.22\n"),
	}

	r := NewCSV(httpResp)
	if r == nil {
		t.Fatal("NewCSV() returned nil")
	}
	if r.CSV() != "symbol,last\nAAPL,150.22\n" {
		t.Errorf("CSV() = %q, want the raw body", r.CSV())
	}
	if r.RateLimit.Limit != 10000 {
		t.Errorf("RateLimit.Limit = %d, want 10000", r.RateLimit.Limit)
	}
}

func TestNewHTML(t *testing.T) {
	httpResp := &internalhttp.Response{
		Raw:     &http.Response{Header: http.Header{}, StatusCode: 200},
		Headers: http.Header{},
		Body:    []byte("<html><body>hi</body></html>"),
	}

	r := NewHTML(httpResp)
	if r == nil {
		t.Fatal("NewHTML() returned nil")
	}
	if r.HTML() != "<html><body>hi</body></html>" {
		t.Errorf("HTML() = %q, want the raw body", r.HTML())
	}
}

func TestNewNoData(t *testing.T) {
	httpResp := &internalhttp.Response{
		Raw: &http.Response{
			Header: http.Header{},
		},
		Headers: http.Header{},
		Body:    []byte{},
	}

	r := NewNoData(httpResp)
	if r == nil {
		t.Fatal("NewNoData() returned nil")
	}
	if !r.NoData {
		t.Error("NoData should be true")
	}
}

func TestParseRateLimitMeta_NilHeaders(t *testing.T) {
	meta := parseRateLimitMeta(nil)
	if meta.Limit != 0 || meta.Remaining != 0 || meta.Consumed != 0 || !meta.ResetAt.IsZero() {
		t.Error("parseRateLimitMeta(nil) should return zero-value RateLimitMeta")
	}
}

func TestParseRateLimitMeta_EmptyHeaders(t *testing.T) {
	meta := parseRateLimitMeta(http.Header{})
	if meta.Limit != 0 || meta.Remaining != 0 || meta.Consumed != 0 || !meta.ResetAt.IsZero() {
		t.Error("parseRateLimitMeta(empty) should return zero-value RateLimitMeta")
	}
}

func TestParseRateLimitMeta_FullHeaders(t *testing.T) {
	headers := http.Header{
		"X-Api-Ratelimit-Limit":     []string{"10000"},
		"X-Api-Ratelimit-Remaining": []string{"9500"},
		"X-Api-Ratelimit-Consumed":  []string{"500"},
		"X-Api-Ratelimit-Reset":     []string{"1704067200"},
	}

	meta := parseRateLimitMeta(headers)
	if meta.Limit != 10000 {
		t.Errorf("Limit = %d, want 10000", meta.Limit)
	}
	if meta.Remaining != 9500 {
		t.Errorf("Remaining = %d, want 9500", meta.Remaining)
	}
	if meta.Consumed != 500 {
		t.Errorf("Consumed = %d, want 500", meta.Consumed)
	}
	expected := time.Unix(1704067200, 0)
	if !meta.ResetAt.Equal(expected) {
		t.Errorf("ResetAt = %v, want %v", meta.ResetAt, expected)
	}
}

func TestParseRateLimitMeta_ZeroReset(t *testing.T) {
	headers := http.Header{
		"X-Api-Ratelimit-Reset": []string{"0"},
	}
	meta := parseRateLimitMeta(headers)
	if !meta.ResetAt.IsZero() {
		t.Error("ResetAt should be zero for reset=0")
	}
}

// TestResponse_Body proves the raw payload is reachable and that the returned
// slice is a copy — a caller mutating it must not corrupt SaveToFile or a
// second read.
func TestResponse_Body(t *testing.T) {
	// want is an immutable string captured before any mutation: comparing
	// against the []byte would alias the same array the test is about to
	// modify, and the aliasing bug would hide itself.
	const want = `{"s":"ok","last":[1.23]}`
	r := &Response{body: []byte(want)}

	got := r.Body()
	if string(got) != want {
		t.Errorf("Body() = %q, want %q", got, want)
	}

	got[0] = 'X'
	if again := r.Body(); string(again) != want {
		t.Errorf("Body() returned an aliased slice: after mutating the caller's copy got %q, want %q", again, want)
	}

	if (&Response{}).Body() != nil {
		t.Error("Body() should be nil when no body was captured")
	}
}

// TestResponse_String pins the summary format and, importantly, that the body
// itself never appears in it: responses land in logs, and a payload may be
// large or carry data the caller would not expect there.
func TestResponse_String(t *testing.T) {
	secret := []byte(`{"account":"private-data"}`)
	r := &Response{
		Response:  &http.Response{Status: "200 OK", StatusCode: 200},
		NoData:    false,
		RateLimit: RateLimitMeta{Limit: 10000, Remaining: 9998},
		body:      secret,
	}

	got := r.String()
	want := "Response{Status: 200 OK, NoData: false, Body: 26 bytes, Credits: 9998/10000}"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if strings.Contains(got, "private-data") {
		t.Errorf("String() leaked the response body: %q", got)
	}

	// A Response with no embedded http.Response must not panic.
	bare := (&Response{NoData: true}).String()
	if !strings.Contains(bare, "no http response") {
		t.Errorf("String() on a bare Response = %q, want a placeholder status", bare)
	}
}

// TestSaveToFile_Permissions pins the mode SaveToFile writes with. A
// response body can hold account-scoped market data, so world-writable is
// not an acceptable default — and a mutation from 0644 to 0666 left the
// whole suite green, because the constant was executed and never compared.
func TestSaveToFile_Permissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.json")
	r := &Response{
		Response: &http.Response{Header: http.Header{}},
		body:     []byte(`{"test": true}`),
	}
	if err := r.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// Compared against the umask-adjusted expectation: the process umask
	// may clear bits, but it can never add them, so a widened mode fails.
	if got := info.Mode().Perm(); got&^0644 != 0 {
		t.Errorf("file mode = %04o, want no bits beyond 0644", got)
	}
	if got := info.Mode().Perm(); got&0002 != 0 {
		t.Errorf("file mode = %04o, want not world-writable", got)
	}
}
