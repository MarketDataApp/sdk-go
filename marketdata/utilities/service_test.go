package utilities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

func newTestService(handler http.Handler) *Service {
	server := httptest.NewServer(handler)
	client := internalhttp.New(internalhttp.Config{
		BaseURL:    server.URL,
		APIVersion: "v1",
		Token:      "test-key",
		RetryCfg:   retry.DefaultConfig(),
		RateLimits: ratelimit.New(),
	})
	return NewService(client)
}

func TestNewService(t *testing.T) {
	client := internalhttp.New(internalhttp.Config{
		BaseURL:    "http://example.com",
		APIVersion: "v1",
		Token:      "test-key",
	})
	svc := NewService(client)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}
}

func TestStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API returns per-service arrays
		resp := apiStatusResponse{
			ResponseStatus: "ok",
			Service:        []string{"api"},
			Status:         []string{"online"},
			Online:         []bool{true},
			UptimePct30d:   []float64{0.9995}, // API returns as decimal, we multiply by 100
			UptimePct90d:   []float64{0.9990},
			Updated:        []int64{1674000000},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	status, _, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Status != "online" {
		t.Errorf("Status = %q, want online", status.Status)
	}
	// API returns 0.9995 which gets multiplied by 100 = 99.95
	if status.Uptime30d != 99.95 {
		t.Errorf("Uptime30d = %f, want 99.95", status.Uptime30d)
	}
	if status.Uptime90d != 99.90 {
		t.Errorf("Uptime90d = %f, want 99.90", status.Uptime90d)
	}
}

func TestStatus_Offline(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API returns per-service arrays with at least one service offline
		resp := apiStatusResponse{
			ResponseStatus: "ok",
			Service:        []string{"api"},
			Status:         []string{"offline"},
			Online:         []bool{false}, // At least one service offline
			UptimePct30d:   []float64{0.9950},
			UptimePct90d:   []float64{0.9980},
			Updated:        []int64{1674000000},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	status, _, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.IsOnline() {
		t.Error("IsOnline() = true, want false for offline status")
	}
}

func TestHeaders(t *testing.T) {
	// Hand-written wire JSON (not a serialized internal struct), matching
	// what the live API actually sends: a flat object, each header name a
	// top-level key — not wrapped in a "headers" envelope field. A prior
	// version of this test serialized the SDK's own (wrong) struct, which
	// is exactly how the wrapper-shape bug went undetected: the mock
	// agreed with the SDK instead of with the API.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Authorization":"Bearer test-****","User-Agent":"MarketData-Go-SDK/2.0.0","Accept":"application/json"}`))
	})

	svc := newTestService(handler)
	headers, _, err := svc.Headers(context.Background())
	if err != nil {
		t.Fatalf("Headers() error = %v", err)
	}

	if len(headers.Headers) != 3 {
		t.Errorf("Headers len = %d, want 3", len(headers.Headers))
	}
	if headers.Headers["User-Agent"] != "MarketData-Go-SDK/2.0.0" {
		t.Errorf("User-Agent = %q, want MarketData-Go-SDK/2.0.0", headers.Headers["User-Agent"])
	}
}

// Test APIStatus methods
func TestAPIStatus_IsOnline(t *testing.T) {
	status := &APIStatus{Status: "online"}
	if !status.IsOnline() {
		t.Error("IsOnline() = false, want true")
	}

	status.Status = "offline"
	if status.IsOnline() {
		t.Error("IsOnline() = true, want false")
	}

	status.Status = "maintenance"
	if status.IsOnline() {
		t.Error("IsOnline() = true, want false for maintenance")
	}
}

// Test error types
func TestValidationError_Error(t *testing.T) {
	err := &sdkerrors.ValidationError{
		Field:   "test",
		Message: "test error",
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &sdkerrors.APIError{
		SupportContext: sdkerrors.SupportContext{Message: "something went wrong"},
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

// Test response conversion
func TestApiStatusResponse_ToAPIStatus_Nil(t *testing.T) {
	var resp *apiStatusResponse
	status := resp.toAPIStatus()
	if status != nil {
		t.Error("toAPIStatus() should return nil for nil response")
	}
}

func TestHeadersResponse_ToHeaders_Nil(t *testing.T) {
	var resp *headersResponse
	headers := resp.toHeaders()
	if headers != nil {
		t.Error("toHeaders() should return nil for nil response")
	}
}

func TestHeadersResponse_ToHeaders_Empty(t *testing.T) {
	var resp headersResponse
	headers := resp.toHeaders()
	if headers == nil {
		t.Fatal("toHeaders() should not return nil for empty response")
	}
	if headers.Headers != nil {
		t.Error("Headers should be nil for a zero-value (unset) response")
	}
}

func TestStatus_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, _, err := svc.Status(context.Background())
	if err == nil {
		t.Fatal("Status() should return error for HTTP error")
	}
}

func TestHeaders_HTTPError(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, _, err := svc.Headers(context.Background())
	if err == nil {
		t.Fatal("Headers() should return error for HTTP error")
	}
}

func TestUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Credit state comes from the x-api-ratelimit-* headers;
		// the body carries the options entitlement and duplicate
		// request counts.
		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "97500")
		w.Header().Set("X-Api-Ratelimit-Consumed", "0")
		w.Header().Set("X-Api-Ratelimit-Reset", "1783776600")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userResponse{
			RequestsRemaining:      97500,
			RequestsLimit:          100000,
			OptionsDataPermissions: "OPRA data delayed 15 minutes",
		})
	})

	svc := newTestService(handler)
	user, _, err := svc.User(context.Background())
	if err != nil {
		t.Fatalf("User() error = %v", err)
	}

	if user.CreditLimit != 100000 {
		t.Errorf("CreditLimit = %d, want 100000", user.CreditLimit)
	}
	if user.CreditsRemaining != 97500 {
		t.Errorf("CreditsRemaining = %d, want 97500", user.CreditsRemaining)
	}
	if user.CreditsConsumed != 0 {
		t.Errorf("CreditsConsumed = %d, want 0", user.CreditsConsumed)
	}
	if user.ResetAt.Unix() != 1783776600 {
		t.Errorf("ResetAt = %v, want unix 1783776600", user.ResetAt)
	}
	if user.OptionsDataPermissions != "OPRA data delayed 15 minutes" {
		t.Errorf("OptionsDataPermissions = %q", user.OptionsDataPermissions)
	}
}

func TestUser_BodyFallbackWithoutHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userResponse{
			RequestsRemaining: 42,
			RequestsLimit:     100,
		})
	})

	svc := newTestService(handler)
	user, _, err := svc.User(context.Background())
	if err != nil {
		t.Fatalf("User() error = %v", err)
	}

	if user.CreditLimit != 100 {
		t.Errorf("CreditLimit = %d, want 100 (body fallback)", user.CreditLimit)
	}
	if user.CreditsRemaining != 42 {
		t.Errorf("CreditsRemaining = %d, want 42 (body fallback)", user.CreditsRemaining)
	}
}

func TestUser_HTTPError(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error", "errmsg": "Invalid token"})
	}))
	_, _, err := svc.User(context.Background())
	if err == nil {
		t.Fatal("User() should return error for HTTP error")
	}
}

func TestUserResponse_ToUserInfo_Nil(t *testing.T) {
	var resp *userResponse
	user := resp.toUserInfo(http.Header{})
	if user != nil {
		t.Error("toUserInfo() should return nil for nil response")
	}
}

func TestUserResponse_ToUserInfo_Empty(t *testing.T) {
	resp := &userResponse{}
	user := resp.toUserInfo(http.Header{})
	if user == nil {
		t.Fatal("toUserInfo() should not return nil for empty response")
	}
	if user.CreditLimit != 0 {
		t.Errorf("CreditLimit = %d, want 0", user.CreditLimit)
	}
	if !user.ResetAt.IsZero() {
		t.Errorf("ResetAt = %v, want zero", user.ResetAt)
	}
}

func TestAPIStatus_String(t *testing.T) {
	s := APIStatus{Status: "online", Uptime30d: 99.95, Uptime90d: 99.90}
	str := s.String()
	if str == "" {
		t.Error("APIStatus.String() returned empty string")
	}
}

func TestHeaders_String(t *testing.T) {
	h := Headers{Headers: map[string]string{"Authorization": "Bearer ***", "User-Agent": "test"}}
	str := h.String()
	if str == "" {
		t.Error("Headers.String() returned empty string")
	}
}

func TestUserInfo_String(t *testing.T) {
	u := UserInfo{CreditLimit: 10000, CreditsRemaining: 9500, OptionsDataPermissions: "OPRA data delayed 15 minutes"}
	str := u.String()
	if str == "" {
		t.Error("UserInfo.String() returned empty string")
	}
}

func TestGetStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiStatusResponse{
			ResponseStatus: "ok",
			Service:        []string{"api"},
			Status:         []string{"online"},
			Online:         []bool{true},
			UptimePct30d:   []float64{0.9995},
			UptimePct90d:   []float64{0.9990},
			Updated:        []int64{1674000000},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	status, err := svc.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status == nil {
		t.Fatal("GetStatus() returned nil")
	}
	if status.Status != "online" {
		t.Errorf("Status = %q, want online", status.Status)
	}
}

func TestGetHeaders(t *testing.T) {
	// Hand-written wire JSON — see TestHeaders for why.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Authorization":"Bearer ****","User-Agent":"test"}`))
	})

	svc := newTestService(handler)
	headers, err := svc.GetHeaders()
	if err != nil {
		t.Fatalf("GetHeaders() error = %v", err)
	}
	if headers == nil {
		t.Fatal("GetHeaders() returned nil")
	}
	if headers.Headers["User-Agent"] != "test" {
		t.Errorf("User-Agent = %q, want test", headers.Headers["User-Agent"])
	}
}

func TestGetUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Api-Ratelimit-Limit", "100000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "99500")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userResponse{
			RequestsRemaining:      99500,
			RequestsLimit:          100000,
			OptionsDataPermissions: "OPRA data delayed 15 minutes",
		})
	})

	svc := newTestService(handler)
	user, err := svc.GetUser()
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if user == nil {
		t.Fatal("GetUser() returned nil")
	}
	if user.CreditsRemaining != 99500 {
		t.Errorf("CreditsRemaining = %d, want 99500", user.CreditsRemaining)
	}
	if user.OptionsDataPermissions == "" {
		t.Error("OptionsDataPermissions should not be empty")
	}
}

func TestStatus_404(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	status, resp, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v, want nil for 404", err)
	}
	if status != nil {
		t.Errorf("Status() returned data, want nil for 404")
	}
	if resp == nil {
		t.Fatal("Status() response = nil, want non-nil NoData response")
	}
}

func TestHeaders_404(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	headers, resp, err := svc.Headers(context.Background())
	if err != nil {
		t.Fatalf("Headers() error = %v, want nil for 404", err)
	}
	// Non-nil and empty: Headers is map-shaped, so the empty map is the
	// correct answer and ranges zero times. Only the scalar-shaped
	// results (Status, User) still come back nil.
	if headers == nil {
		t.Fatal("Headers() returned nil for a markerless 404; want an empty result")
	}
	if len(headers.Headers) != 0 {
		t.Errorf("Headers() headers = %d, want 0", len(headers.Headers))
	}
	for range headers.Headers { // must not panic
		t.Error("unreachable")
	}
	if resp == nil {
		t.Fatal("Headers() response = nil, want non-nil NoData response")
	}
	if !resp.NoData {
		t.Error("Headers() response NoData = false, want true")
	}
}

func TestUser_404(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	user, resp, err := svc.User(context.Background())
	if err != nil {
		t.Fatalf("User() error = %v, want nil for 404", err)
	}
	if user != nil {
		t.Errorf("User() returned data, want nil for 404")
	}
	if resp == nil {
		t.Fatal("User() response = nil, want non-nil NoData response")
	}
}

// TestStatus_ErrorBodyIsNotOnline pins the guard that was missing: a 200
// response whose body reports its own failure must be an error, not an
// APIStatus claiming the API is up. This is the monitoring endpoint, so
// answering "online" to {"s":"error"} is the worst possible default.
func TestStatus_ErrorBodyIsNotOnline(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
	}{
		{"explicit error", `{"s":"error","errmsg":"Internal Server Error"}`},
		{"empty object", `{}`},
		{"ok but no services listed", `{"s":"ok"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))

			st, _, err := svc.Status(context.Background())
			if tt.name == "ok but no services listed" {
				// "ok" with no per-service array decodes, but must not claim
				// health it has no evidence for.
				if err != nil {
					t.Fatalf("Status() error = %v, want nil for an ok body", err)
				}
				if st.IsOnline() {
					t.Errorf("IsOnline() = true with no services listed, want false")
				}
				return
			}
			if err == nil {
				t.Fatalf("Status() error = nil, want an error for body %s", tt.body)
			}
			if st != nil {
				t.Errorf("Status() = %+v, want nil alongside an error", st)
			}
		})
	}
}

// TestToUserInfo_EntitlementsFromHeader pins the header fallback. The body
// key exists but production always sends it empty, so reading only the body
// reported no entitlements for an account that has them all.
func TestToUserInfo_EntitlementsFromHeader(t *testing.T) {
	const live = "delayed_quotes_permission,historical_quotes_permission,real_time_quotes_permission"

	t.Run("header fills an empty body", func(t *testing.T) {
		r := &userResponse{OptionsDataPermissions: ""}
		h := http.Header{"X-Options-Data-Permissions": []string{live}}
		if got := r.toUserInfo(h).OptionsDataPermissions; got != live {
			t.Errorf("OptionsDataPermissions = %q, want the header value", got)
		}
	})

	t.Run("body wins when the header is absent", func(t *testing.T) {
		r := &userResponse{OptionsDataPermissions: "realtime"}
		if got := r.toUserInfo(http.Header{}).OptionsDataPermissions; got != "realtime" {
			t.Errorf("OptionsDataPermissions = %q, want realtime", got)
		}
	})

	t.Run("neither source", func(t *testing.T) {
		r := &userResponse{}
		if got := r.toUserInfo(http.Header{}).OptionsDataPermissions; got != "" {
			t.Errorf("OptionsDataPermissions = %q, want empty", got)
		}
	})
}
