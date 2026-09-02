package funds

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCandles(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := candlesResponse{
			Status: "ok",
			Time:   []int64{1674000000, 1674086400, 1674172800},
			Open:   []float64{100.0, 101.0, 102.0},
			High:   []float64{101.0, 102.0, 103.0},
			Low:    []float64{99.0, 100.0, 101.0},
			Close:  []float64{100.5, 101.5, 102.5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	candles, _, err := svc.Candles(context.Background(), "SPY")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	if len(candles) != 3 {
		t.Fatalf("Candles len = %d, want 3", len(candles))
	}
	if candles[0].Open != 100.0 {
		t.Errorf("candles[0].Open = %f, want 100.0", candles[0].Open)
	}
	if candles[0].Close != 100.5 {
		t.Errorf("candles[0].Close = %f, want 100.5", candles[0].Close)
	}
}

func TestCandles_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL path contains resolution
		if r.URL.Path != "/v1/funds/candles/W/SPY/" {
			t.Errorf("path = %q, want /v1/funds/candles/W/SPY/", r.URL.Path)
		}

		// Verify query parameters
		if r.URL.Query().Get("countback") != "5" {
			t.Errorf("countback = %q, want 5", r.URL.Query().Get("countback"))
		}
		if r.URL.Query().Get("from") != "" {
			t.Errorf("from should be unset for a countback window, got %q", r.URL.Query().Get("from"))
		}

		resp := candlesResponse{
			Status: "ok",
			Time:   []int64{1674000000},
			Open:   []float64{100.0},
			High:   []float64{101.0},
			Low:    []float64{99.0},
			Close:  []float64{100.5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)

	_, _, err := svc.Candles(context.Background(), "SPY",
		WithResolution(ResolutionWeekly),
		WithCandleWindow(LastN(5)),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

func TestCandles_Between(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "2024-01-01" {
			t.Errorf("from = %q, want 2024-01-01", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "2024-01-31" {
			t.Errorf("to = %q, want 2024-01-31", r.URL.Query().Get("to"))
		}

		resp := candlesResponse{Status: "ok"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Candles(context.Background(), "SPY", WithCandleWindow(Between(from, to)))
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
}

// TestCandles_OnDate proves the OnDate window reaches the API as a single
// date= parameter — the mode funds candles could not previously express.
func TestCandles_OnDate(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") != "2024-01-05" {
			t.Errorf("date = %q, want 2024-01-05", r.URL.Query().Get("date"))
		}
		if r.URL.Query().Get("from") != "" || r.URL.Query().Get("to") != "" {
			t.Error("from/to must be unset for an OnDate window")
		}

		resp := candlesResponse{
			Status: "ok",
			Time:   []int64{1704430800},
			Open:   []float64{433.44},
			High:   []float64{433.44},
			Low:    []float64{433.44},
			Close:  []float64{433.44},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	candles, _, err := svc.Candles(context.Background(), "VFINX",
		WithCandleWindow(OnDate(time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC))),
	)
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("Candles len = %d, want 1", len(candles))
	}
}

// TestCandles_InvalidWindow verifies an invalid window is rejected before any
// network request is made.
func TestCandles_InvalidWindow(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be reached for an invalid window")
	})

	svc := newTestService(handler)
	from := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Candles(context.Background(), "SPY", WithCandleWindow(Between(from, to)))
	if err == nil {
		t.Fatal("Candles() should return a validation error for from after to")
	}
	if _, ok := err.(*sdkerrors.ValidationError); !ok {
		t.Fatalf("error type = %T, want *sdkerrors.ValidationError", err)
	}
}

func TestCandles_EmptySymbol(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Candles(context.Background(), "")
	if err == nil {
		t.Fatal("Candles() should return error for empty symbol")
	}
	valErr, ok := err.(*sdkerrors.ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *sdkerrors.ValidationError", err)
	}
	if valErr.Field != "symbol" {
		t.Errorf("Field = %q, want symbol", valErr.Field)
	}
}

func TestCandles_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error", "errmsg": "something went wrong"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Candles(context.Background(), "SPY")
	if err == nil {
		t.Fatal("Candles() should return error for bad status")
	}
}

// Test type methods
func TestCandle_Range(t *testing.T) {
	candle := Candle{
		High: 105.0,
		Low:  100.0,
	}
	r := candle.Range()
	if r != 5.0 {
		t.Errorf("Range() = %f, want 5.0", r)
	}
}

func TestResolution_String(t *testing.T) {
	tests := []struct {
		res  Resolution
		want string
	}{
		{ResolutionDaily, "D"},
		{ResolutionWeekly, "W"},
		{ResolutionMonthly, "M"},
		{ResolutionYearly, "Y"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.res.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test error types
func TestValidationError_Error(t *testing.T) {
	err := &sdkerrors.ValidationError{
		Field:   "symbol",
		Message: "symbol is required",
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
func TestCandlesResponse_ToCandles_Nil(t *testing.T) {
	var resp *candlesResponse
	candles := resp.toCandles()
	if candles != nil {
		t.Error("toCandles() should return nil for nil response")
	}
}

func TestCandlesResponse_ToCandles_Empty(t *testing.T) {
	resp := &candlesResponse{}
	candles := resp.toCandles()
	if candles != nil {
		t.Error("toCandles() should return nil for empty response")
	}
}

// Test safe index
func TestSafeIndex(t *testing.T) {
	slice := []float64{1.0, 2.0, 3.0}
	if safeIndex(slice, 0) != 1.0 {
		t.Error("safeIndex(0) should return 1.0")
	}
	if safeIndex(slice, 5) != 0 {
		t.Error("safeIndex(5) should return 0 for out of bounds")
	}
}

// Test options
func TestDefaultCandleOptions(t *testing.T) {
	opts := defaultCandleOptions()
	if opts.resolution != ResolutionDaily {
		t.Errorf("default resolution = %q, want D", opts.resolution)
	}
}

func TestCandles_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Candles(context.Background(), "SPY")
	if err == nil {
		t.Fatal("Candles() should return error for HTTP error")
	}
}

func TestCandle_String(t *testing.T) {
	c := Candle{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 95, Close: 102}
	s := c.String()
	if s == "" {
		t.Error("Candle.String() returned empty string")
	}
}

func TestGetCandles(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := candlesResponse{
			Status: "ok",
			Time:   []int64{1704067200},
			Open:   []float64{150.0},
			High:   []float64{155.0},
			Low:    []float64{149.0},
			Close:  []float64{153.0},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	candles, err := svc.GetCandles("SPY")
	if err != nil {
		t.Fatalf("GetCandles() error = %v", err)
	}
	if len(candles) != 1 {
		t.Fatalf("GetCandles len = %d, want 1", len(candles))
	}
	if candles[0].Open != 150.0 {
		t.Errorf("candles[0].Open = %f, want 150.0", candles[0].Open)
	}
	if candles[0].High != 155.0 {
		t.Errorf("candles[0].High = %f, want 155.0", candles[0].High)
	}
	if candles[0].Low != 149.0 {
		t.Errorf("candles[0].Low = %f, want 149.0", candles[0].Low)
	}
	if candles[0].Close != 153.0 {
		t.Errorf("candles[0].Close = %f, want 153.0", candles[0].Close)
	}
}

func TestCandles_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	svc := newTestService(handler)
	candles, resp, err := svc.Candles(context.Background(), "INVALID")
	if err != nil {
		t.Fatalf("Candles() error = %v, want nil for 404", err)
	}
	if candles != nil {
		t.Errorf("Candles() returned data, want nil for 404")
	}
	if resp == nil {
		t.Fatal("Candles() response = nil, want non-nil NoData response")
	}
}

func TestCandles_PathInjection(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "t": []int64{}, "o": []float64{}, "h": []float64{}, "l": []float64{}, "c": []float64{}})
	})

	svc := newTestService(handler)
	_, _, err := svc.Candles(context.Background(), "VFINX/../../user")
	if err != nil {
		t.Fatalf("Candles() error = %v", err)
	}

	if !strings.Contains(gotPath, "VFINX%2F..%2F..%2Fuser") {
		t.Errorf("request path = %q, want escaped symbol segment", gotPath)
	}
}
