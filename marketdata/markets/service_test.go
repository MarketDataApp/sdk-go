package markets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		// API doesn't return 'open' bool array, we derive from status string
		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000},
			Stat:   []string{"open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	status, _, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !status.Open {
		t.Error("Open = false, want true")
	}
	if status.Status != "open" {
		t.Errorf("Status = %q, want open", status.Status)
	}
}

func TestStatus_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") == "" {
			t.Error("date parameter should be set")
		}
		if r.URL.Query().Get("country") != "US" {
			t.Errorf("country = %q, want US", r.URL.Query().Get("country"))
		}

		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000},
			Stat:   []string{"open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Status(context.Background(),
		WithDate(date),
		WithCountry("US"),
	)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
}

func TestStatus_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Status(context.Background())
	if err == nil {
		t.Fatal("Status() should return error for bad status")
	}
}

func TestStatusHistory(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API doesn't return 'open' bool array, we derive from status string
		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000, 1674086400, 1674172800},
			Stat:   []string{"open", "closed", "open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	statuses, _, err := svc.StatusHistory(context.Background(),
		WithHistoryWindow(Between(from, to)),
	)
	if err != nil {
		t.Fatalf("StatusHistory() error = %v", err)
	}

	if len(statuses) != 3 {
		t.Fatalf("StatusHistory len = %d, want 3", len(statuses))
	}
	if !statuses[0].Open {
		t.Error("statuses[0].Open = false, want true")
	}
	if statuses[1].Open {
		t.Error("statuses[1].Open = true, want false")
	}
}

func TestStatusHistory_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.StatusHistory(context.Background())
	if err == nil {
		t.Fatal("StatusHistory() should return error for bad status")
	}
}

// Test MarketStatus methods
func TestMarketStatus_OpenClosedHelpers(t *testing.T) {
	// Three honest states, keyed off the API's status string: "open",
	// "closed", and empty (a day outside the calendar's coverage), which
	// must read as neither open nor closed.
	cases := []struct {
		status           string
		isOpen, isClosed bool
	}{
		{"open", true, false},
		{"closed", false, true},
		{"", false, false},
	}
	for _, tc := range cases {
		m := &MarketStatus{Status: tc.status, Open: tc.status == "open"}
		if got := m.IsOpen(); got != tc.isOpen {
			t.Errorf("IsOpen() with status %q = %t, want %t", tc.status, got, tc.isOpen)
		}
		if got := m.IsClosed(); got != tc.isClosed {
			t.Errorf("IsClosed() with status %q = %t, want %t", tc.status, got, tc.isClosed)
		}
	}
}

// Test error types
func TestValidationError_Error(t *testing.T) {
	err := &sdkerrors.ValidationError{
		Field:   "date",
		Message: "invalid date",
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
func TestStatusResponse_ToMarketStatus_Nil(t *testing.T) {
	var resp *statusResponse
	status := resp.toMarketStatus()
	if status != nil {
		t.Error("toMarketStatus() should return nil for nil response")
	}
}

func TestStatusResponse_ToMarketStatus_Empty(t *testing.T) {
	resp := &statusResponse{}
	status := resp.toMarketStatus()
	if status != nil {
		t.Error("toMarketStatus() should return nil for empty response")
	}
}

func TestStatusResponse_ToMarketStatuses_Nil(t *testing.T) {
	var resp *statusResponse
	statuses := resp.toMarketStatuses()
	if statuses != nil {
		t.Error("toMarketStatuses() should return nil for nil response")
	}
}

func TestStatusResponse_ToMarketStatuses_Empty(t *testing.T) {
	resp := &statusResponse{}
	statuses := resp.toMarketStatuses()
	if statuses != nil {
		t.Error("toMarketStatuses() should return nil for empty response")
	}
}

// Test safe index functions
func TestSafeIndexStr(t *testing.T) {
	slice := []string{"open", "closed", "early-close"}
	if safeIndexStr(slice, 0) != "open" {
		t.Error("safeIndexStr(0) should return 'open'")
	}
	if safeIndexStr(slice, 5) != "" {
		t.Error("safeIndexStr(5) should return empty string for out of bounds")
	}
}

func TestStatus_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Status(context.Background())
	if err == nil {
		t.Fatal("Status() should return error for HTTP error")
	}
}

func TestStatusHistory_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.StatusHistory(context.Background())
	if err == nil {
		t.Fatal("StatusHistory() should return error for HTTP error")
	}
}

func TestStatusHistory_WithCountry(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("country") != "US" {
			t.Errorf("country = %q, want US", r.URL.Query().Get("country"))
		}

		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000},
			Stat:   []string{"open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.StatusHistory(context.Background(), WithCountry("US"))
	if err != nil {
		t.Fatalf("StatusHistory() error = %v", err)
	}
}

func TestStatusHistory_WithCountback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("countback") != "5" {
			t.Errorf("countback = %q, want 5", r.URL.Query().Get("countback"))
		}

		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000, 1674086400, 1674172800, 1674259200, 1674345600},
			Stat:   []string{"open", "closed", "open", "open", "closed"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	statuses, _, err := svc.StatusHistory(context.Background(), WithHistoryWindow(LastN(5)))
	if err != nil {
		t.Fatalf("StatusHistory() error = %v", err)
	}
	if len(statuses) != 5 {
		t.Fatalf("StatusHistory len = %d, want 5", len(statuses))
	}
}

func TestMarketStatus_String(t *testing.T) {
	m := MarketStatus{Date: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Status: "open", Open: true}
	s := m.String()
	if s == "" {
		t.Error("MarketStatus.String() returned empty string")
	}
}

func TestGetStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := statusResponse{
			Status: "ok",
			Stat:   []string{"open"},
			Date:   []int64{1704067200},
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
	if !status.Open {
		t.Error("Open = false, want true")
	}
	if status.Status != "open" {
		t.Errorf("Status = %q, want open", status.Status)
	}
}

func TestStatus_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	svc := newTestService(handler)
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

func TestStatusHistory_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	svc := newTestService(handler)
	statuses, resp, err := svc.StatusHistory(context.Background())
	if err != nil {
		t.Fatalf("StatusHistory() error = %v, want nil for 404", err)
	}
	if statuses != nil {
		t.Errorf("StatusHistory() returned data, want nil for 404")
	}
	if resp == nil {
		t.Fatal("StatusHistory() response = nil, want non-nil NoData response")
	}
}

// TestHistoryWindow_QueryParams asserts that each HistoryWindow mode
// serializes to exactly the query parameters the API expects. Because a
// HistoryWindow is a single sealed value, only one mode's parameters can ever
// appear in a request.
func TestHistoryWindow_QueryParams(t *testing.T) {
	from := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		window HistoryWindow
		want   map[string]string
	}{
		{
			name:   "Between",
			window: Between(from, to),
			want:   map[string]string{"from": "2024-01-02", "to": "2024-01-10"},
		},
		{
			name:   "Since",
			window: Since(from),
			want:   map[string]string{"from": "2024-01-02"},
		},
		{
			name:   "Until",
			window: Until(to),
			want:   map[string]string{"to": "2024-01-10"},
		},
		{
			name:   "LastN",
			window: LastN(5),
			want:   map[string]string{"countback": "5"},
		},
		{
			name:   "LastNUntil",
			window: LastNUntil(3, to),
			want:   map[string]string{"countback": "3", "to": "2024-01-10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := r.URL.Query()
				// Every expected param must be present with the exact value.
				for k, v := range tt.want {
					if got.Get(k) != v {
						t.Errorf("query[%q] = %q, want %q (full query %q)", k, got.Get(k), v, r.URL.RawQuery)
					}
				}
				// No range param outside the expected set may appear.
				for _, k := range []string{"date", "from", "to", "countback"} {
					if _, ok := tt.want[k]; !ok && got.Has(k) {
						t.Errorf("unexpected query param %q = %q", k, got.Get(k))
					}
				}

				resp := statusResponse{
					Status: "ok",
					Date:   []int64{1674000000},
					Stat:   []string{"open"},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			})

			svc := newTestService(handler)
			_, _, err := svc.StatusHistory(context.Background(), WithHistoryWindow(tt.window))
			if err != nil {
				t.Fatalf("StatusHistory() error = %v", err)
			}
		})
	}
}

// TestWithCountry_BothMethods confirms that the single WithCountry option
// compiles and applies for both Service.Status (a StatusOption context) and
// Service.StatusHistory (a HistoryOption context). This is the shared-option
// mechanism: CountryOption satisfies both option interfaces.
func TestWithCountry_BothMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("country") != "CA" {
			t.Errorf("country = %q, want CA", r.URL.Query().Get("country"))
		}
		resp := statusResponse{
			Status: "ok",
			Date:   []int64{1674000000},
			Stat:   []string{"open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)

	// The exact same option value is accepted by both methods.
	country := WithCountry("CA")

	if _, _, err := svc.Status(context.Background(), country); err != nil {
		t.Fatalf("Status() with WithCountry error = %v", err)
	}
	if _, _, err := svc.StatusHistory(context.Background(), country, WithHistoryWindow(LastN(1))); err != nil {
		t.Fatalf("StatusHistory() with WithCountry error = %v", err)
	}
}

// TestStatusHistory_WindowValidation confirms that an invalid HistoryWindow is
// rejected before any request is made to the API.
func TestStatusHistory_WindowValidation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for an invalid window")
	})

	svc := newTestService(handler)
	from := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC) // to before from

	_, _, err := svc.StatusHistory(context.Background(),
		WithHistoryWindow(Between(from, to)),
	)
	if err == nil {
		t.Fatal("StatusHistory() should return a validation error for from after to")
	}
	var verr *sdkerrors.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T, want *sdkerrors.ValidationError", err)
	}
}

func TestGetStatusHistory(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := statusResponse{
			Status: "ok",
			Stat:   []string{"closed", "open"},
			Date:   []int64{1704067200, 1704153600},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	hist, err := svc.GetStatusHistory()
	if err != nil {
		t.Fatalf("GetStatusHistory() error = %v", err)
	}
	if len(hist) != 2 {
		t.Fatalf("len = %d, want 2", len(hist))
	}
	if hist[0].Open || !hist[1].Open {
		t.Errorf("statuses = %v/%v, want closed then open", hist[0].Open, hist[1].Open)
	}
}
