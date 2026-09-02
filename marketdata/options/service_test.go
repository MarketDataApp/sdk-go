package options

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/ratelimit"
	"github.com/MarketDataApp/sdk-go/v2/internal/retry"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
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

func TestChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"AAPL230120C00150000"},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.50},
			BidSize:      []int{100},
			Ask:          []float64{5.60},
			AskSize:      []int{50},
			Last:         []float64{5.55},
			Volume:       []int64{1000},
			OpenInterest: []int64{5000},
			IV:           []float64{0.25},
			Delta:        []float64{0.55},
			Gamma:        []float64{0.02},
			Theta:        []float64{-0.05},
			Vega:         []float64{0.15},
			InTheMoney:   []bool{true},
			Updated:      []int64{1674100000},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	chain, _, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	if chain.Underlying != "AAPL" {
		t.Errorf("Underlying = %q, want AAPL", chain.Underlying)
	}
	if len(chain.Options) != 1 {
		t.Fatalf("Options len = %d, want 1", len(chain.Options))
	}
	if chain.Options[0].OptionSymbol != "AAPL230120C00150000" {
		t.Errorf("OptionSymbol = %q, want AAPL230120C00150000", chain.Options[0].OptionSymbol)
	}
	if chain.Options[0].Strike != 150.0 {
		t.Errorf("Strike = %f, want 150.0", chain.Options[0].Strike)
	}
	if chain.Options[0].Type != Call {
		t.Errorf("Type = %q, want call", chain.Options[0].Type)
	}
	if !chain.Options[0].InTheMoney {
		t.Error("InTheMoney = false, want true")
	}
}

func TestChain_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("side") != "call" {
			t.Errorf("side = %q, want call", r.URL.Query().Get("side"))
		}
		if r.URL.Query().Get("strike") != "150-160" {
			t.Errorf("strike = %q, want 150-160", r.URL.Query().Get("strike"))
		}
		if r.URL.Query().Get("dte") != "30" {
			t.Errorf("dte = %q, want 30", r.URL.Query().Get("dte"))
		}

		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL",
		WithSide(SideCall),
		WithStrike(StrikeRange(150, 160)),
		WithExpiry(InDTE(30)),
	)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

// TestChain_ExpirationSelector verifies the expiration mode of ExpiryFilter
// serializes to the expiration parameter.
func TestChain_ExpirationSelector(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("expiration") != "2024-01-20" {
			t.Errorf("expiration = %q, want 2024-01-20", r.URL.Query().Get("expiration"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{Status: "ok", OptionSymbol: []string{}})
	})

	svc := newTestService(handler)
	expDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpiry(OnExpiration(expDate)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_EmptySymbol(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Chain(context.Background(), "")
	if err == nil {
		t.Fatal("Chain() should return error for empty symbol")
	}
	valErr, ok := err.(*sdkerrors.ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if valErr.Field != "symbol" {
		t.Errorf("Field = %q, want symbol", valErr.Field)
	}
}

func TestChain_WithAllParams(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// Verify key params
		if q.Get("date") != "2024-01-15" {
			t.Errorf("date = %q, want 2024-01-15", q.Get("date"))
		}
		if q.Get("month") != "3" {
			t.Errorf("month = %q, want 3", q.Get("month"))
		}
		if q.Get("year") != "2024" {
			t.Errorf("year = %q, want 2024", q.Get("year"))
		}
		if q.Get("weekly") != "true" {
			t.Errorf("weekly = %q, want true", q.Get("weekly"))
		}
		if q.Get("monthly") != "true" {
			t.Errorf("monthly = %q, want true", q.Get("monthly"))
		}
		if q.Get("delta") != "0.5" {
			t.Errorf("delta = %q, want 0.5", q.Get("delta"))
		}
		if q.Get("strikeLimit") != "10" {
			t.Errorf("strikeLimit = %q, want 10", q.Get("strikeLimit"))
		}
		if q.Get("range") != "itm" {
			t.Errorf("range = %q, want itm", q.Get("range"))
		}
		if q.Get("minBid") != "0.5" {
			t.Errorf("minBid = %q, want 0.5", q.Get("minBid"))
		}
		if q.Get("maxBidAskSpread") != "1.5" {
			t.Errorf("maxBidAskSpread = %q, want 1.5", q.Get("maxBidAskSpread"))
		}
		if q.Get("minOpenInterest") != "100" {
			t.Errorf("minOpenInterest = %q, want 100", q.Get("minOpenInterest"))
		}
		if q.Get("minVolume") != "50" {
			t.Errorf("minVolume = %q, want 50", q.Get("minVolume"))
		}
		if q.Get("nonstandard") != "false" {
			t.Errorf("nonstandard = %q, want false", q.Get("nonstandard"))
		}

		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Chain(context.Background(), "AAPL",
		WithChainDate(date),
		WithExpiry(InMonthOfYear(3, 2024)),
		WithExpirationTypes(IncludeExpirationTypes(Weekly, Monthly)),
		WithStrike(ByDelta(0.5)),
		WithStrikeLimit(10),
		WithRange(MoneynessITM),
		WithMinBid(0.5),
		WithMaxBidAskSpread(1.5),
		WithMinOpenInterest(100),
		WithMinVolume(50),
		WithNonstandard(false),
	)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_StrikeMinExpression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("strike") != ">=140" {
			t.Errorf("strike = %q, want >=140", r.URL.Query().Get("strike"))
		}

		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL",
		WithStrike(MinStrike(140)),
	)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_StrikeExprPassthrough(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("strike") != "150" {
			t.Errorf("strike = %q, want 150", r.URL.Query().Get("strike"))
		}

		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL",
		WithStrike(StrikeExpr("150")),
	)
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error", "errmsg": "something went wrong"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Chain() should return error for bad status")
	}
}

func TestExpirations(t *testing.T) {
	// Hand-written wire JSON (not a serialized internal struct) so a struct
	// tag typo cannot silently hide from the test.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","expirations":[1674172800,1676764800,1679443200],"updated":1674100000}`))
	})

	svc := newTestService(handler)
	exps, _, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if exps == nil {
		t.Fatal("Expirations() returned nil")
	}

	if len(exps.Dates) != 3 {
		t.Fatalf("Dates len = %d, want 3", len(exps.Dates))
	}
	// The response-level updated field must surface, normalized to Eastern.
	if want := timezone.ToEastern(1674100000); !exps.Updated.Equal(want) {
		t.Errorf("Updated = %v, want %v", exps.Updated, want)
	}
}

func TestExpirations_UpdatedOmitted(t *testing.T) {
	// Without a wire updated field the timestamp stays the zero time (so
	// IsZero works), never the Unix epoch.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","expirations":[1674172800]}`))
	})

	svc := newTestService(handler)
	exps, _, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if exps == nil {
		t.Fatal("Expirations() returned nil")
	}
	if !exps.Updated.IsZero() {
		t.Errorf("Updated = %v, want zero time when the API omits updated", exps.Updated)
	}
}

func TestExpirations_EmptyListIsNotNoData(t *testing.T) {
	// A 200 OK with an empty list is real data (currently no expirations
	// listed), distinct from a 404. The doc promises a nil *Expirations only
	// on 404 (checked via Response.NoData) — a non-nil pointer with an empty
	// Dates slice here is required for that contract to hold.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","expirations":[]}`))
	})

	svc := newTestService(handler)
	exps, resp, err := svc.Expirations(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
	if exps == nil {
		t.Fatal("Expirations() returned nil for a 200 OK empty list, want a non-nil *Expirations with empty Dates")
	}
	if len(exps.Dates) != 0 {
		t.Errorf("Dates len = %d, want 0", len(exps.Dates))
	}
	if resp.NoData {
		t.Error("NoData = true for a 200 OK response, want false")
	}
}

func TestExpirations_String(t *testing.T) {
	e := Expirations{
		Dates:   []time.Time{time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)},
		Updated: time.Date(2023, 1, 19, 4, 26, 40, 0, time.UTC),
	}
	s := e.String()
	if s == "" {
		t.Error("Expirations.String() returned empty string")
	}
	if want := "2023-01-20"; !strings.Contains(s, want) {
		t.Errorf("String() = %q, missing next expiration %q", s, want)
	}
	empty := Expirations{}
	if empty.String() == "" {
		t.Error("Expirations.String() returned empty string for empty list")
	}
}

func TestExpirations_WithOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("strike") != "150" {
			t.Errorf("strike = %q, want 150", r.URL.Query().Get("strike"))
		}
		if r.URL.Query().Get("date") == "" {
			t.Error("date parameter should be set")
		}

		resp := expirationsResponse{
			Status:      "ok",
			Expirations: []int64{1674172800},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Expirations(context.Background(), "AAPL",
		WithExpirationStrike(150),
		WithExpirationDate(date),
	)
	if err != nil {
		t.Fatalf("Expirations() error = %v", err)
	}
}

func TestExpirations_EmptySymbol(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Expirations(context.Background(), "")
	if err == nil {
		t.Fatal("Expirations() should return error for empty symbol")
	}
}

func TestExpirations_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Expirations(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Expirations() should return error for bad status")
	}
}

func TestQuote(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"AAPL230120C00150000"},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.50},
			BidSize:      []int{100},
			Ask:          []float64{5.60},
			AskSize:      []int{50},
			Last:         []float64{5.55},
			Volume:       []int64{1000},
			OpenInterest: []int64{5000},
			IV:           []float64{0.25},
			Delta:        []float64{0.55},
			Gamma:        []float64{0.02},
			Theta:        []float64{-0.05},
			Vega:         []float64{0.15},
			Updated:      []int64{1674100000},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	quote, _, err := svc.Quote(context.Background(), "AAPL230120C00150000")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	if quote.OptionSymbol != "AAPL230120C00150000" {
		t.Errorf("OptionSymbol = %q, want AAPL230120C00150000", quote.OptionSymbol)
	}
	if quote.Strike != 150.0 {
		t.Errorf("Strike = %f, want 150.0", quote.Strike)
	}
	if quote.Delta != 0.55 {
		t.Errorf("Delta = %f, want 0.55", quote.Delta)
	}
}

func TestQuote_WithDateOptions(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") != "2024-01-15" {
			t.Errorf("date = %q, want 2024-01-15", r.URL.Query().Get("date"))
		}

		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"AAPL230120C00150000"},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.50},
			Ask:          []float64{5.60},
			Mid:          []float64{5.55},
			Last:         []float64{5.55},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	quote, _, err := svc.Quote(context.Background(), "AAPL230120C00150000",
		WithOptionQuoteWindow(QuoteOnDate(date)),
	)
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.Mid != 5.55 {
		t.Errorf("Mid = %f, want 5.55", quote.Mid)
	}
}

func TestQuote_EmptySymbol(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Quote(context.Background(), "")
	if err == nil {
		t.Fatal("Quote() should return error for empty symbol")
	}
	valErr, ok := err.(*sdkerrors.ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if valErr.Field != "optionSymbol" {
		t.Errorf("Field = %q, want optionSymbol", valErr.Field)
	}
}

func TestQuote_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Quote(context.Background(), "AAPL230120C00150000")
	if err == nil {
		t.Fatal("Quote() should return error for bad status")
	}
}

func TestLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The whole human-readable query travels as one escaped path
		// segment; the API does not accept query parameters.
		want := "/v1/options/lookup/AAPL%202024-01-20%20150%20call/"
		if r.URL.EscapedPath() != want {
			t.Errorf("path = %q, want %q", r.URL.EscapedPath(), want)
		}

		resp := lookupResponse{
			Status:       "ok",
			OptionSymbol: "AAPL230120C00150000",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	expDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	symbol, _, err := svc.Lookup(context.Background(), "AAPL", expDate, 150.0, Call)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}

	if symbol != "AAPL230120C00150000" {
		t.Errorf("symbol = %q, want AAPL230120C00150000", symbol)
	}
}

func TestLookup_EmptyUnderlying(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.Lookup(context.Background(), "", time.Time{}, 150.0, Call)
	if err == nil {
		t.Fatal("Lookup() should return error for empty underlying")
	}
	valErr, ok := err.(*sdkerrors.ValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	if valErr.Field != "underlying" {
		t.Errorf("Field = %q, want underlying", valErr.Field)
	}
}

// TestLookup_RejectsZeroArguments pins that an unset argument is caught before
// the request. Interpolated verbatim they produce a query like
// "AAPL 0001-01-01 0 ", whose 404 is reported as no-data — so without these
// checks a malformed call is indistinguishable from a contract that does not
// exist, and still costs a request.
func TestLookup_RejectsZeroArguments(t *testing.T) {
	validExp := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	requestMade := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
		w.WriteHeader(http.StatusNotFound)
	})
	svc := newTestService(handler)

	cases := []struct {
		name       string
		underlying string
		expiration time.Time
		strike     float64
		optionType OptionType
		wantField  string
	}{
		{"ZeroExpiration", "AAPL", time.Time{}, 150, Call, "expiration"},
		{"ZeroStrike", "AAPL", validExp, 0, Call, "strike"},
		{"NegativeStrike", "AAPL", validExp, -150, Call, "strike"},
		{"EmptyType", "AAPL", validExp, 150, "", "optionType"},
		{"UnknownType", "AAPL", validExp, 150, OptionType("straddle"), "optionType"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sym, resp, err := svc.Lookup(context.Background(), tc.underlying, tc.expiration, tc.strike, tc.optionType)
			var valErr *sdkerrors.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("error = %v (%T), want *sdkerrors.ValidationError", err, err)
			}
			if valErr.Field != tc.wantField {
				t.Errorf("Field = %q, want %q", valErr.Field, tc.wantField)
			}
			if sym != "" || resp != nil {
				t.Errorf("got (%q, %v), want (\"\", nil)", sym, resp)
			}
		})
	}
	if requestMade {
		t.Error("a request reached the server; validation must happen pre-network")
	}
}

func TestLookup_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"s": "error"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	_, _, err := svc.Lookup(context.Background(), "AAPL", time.Now(), 150.0, Call)
	if err == nil {
		t.Fatal("Lookup() should return error for bad status")
	}
}

// Test helper methods
func TestOptionQuote_Spread(t *testing.T) {
	contract := OptionQuote{
		Bid: 5.50,
		Ask: 5.60,
	}
	spread := contract.Spread()
	expected := 0.10
	if spread < expected-0.001 || spread > expected+0.001 {
		t.Errorf("Spread() = %f, want %f", spread, expected)
	}
}

func TestOptionQuote_CalcMid(t *testing.T) {
	contract := OptionQuote{
		Bid: 5.50,
		Ask: 5.60,
	}
	mid := contract.CalcMid()
	expected := 5.55
	if mid != expected {
		t.Errorf("Mid() = %f, want %f", mid, expected)
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

// Test response conversion methods
func TestChainResponse_ToOptionsChain_Nil(t *testing.T) {
	var resp *quoteResponse
	chain := resp.toOptionsChain()
	if chain == nil || len(chain.Options) != 0 {
		t.Error("toOptionsChain() should return empty chain for nil response")
	}
}

func TestChainResponse_ToOptionsChain_Empty(t *testing.T) {
	resp := &quoteResponse{}
	chain := resp.toOptionsChain()
	if chain == nil || len(chain.Options) != 0 {
		t.Error("toOptionsChain() should return empty chain for empty response")
	}
}

func TestExpirationsResponse_ToExpirations_Nil(t *testing.T) {
	var resp *expirationsResponse
	exps := resp.toExpirations()
	if exps != nil {
		t.Error("toExpirations() should return nil for nil response")
	}
}

func TestExpirationsResponse_ToExpirations_Empty(t *testing.T) {
	resp := &expirationsResponse{}
	exps := resp.toExpirations()
	if exps == nil || len(exps.Dates) != 0 {
		t.Error("toExpirations() should return a non-nil Expirations with an empty Dates slice for a 200 with no expirations — nil is reserved for 404 (NoData)")
	}
}

func TestQuoteResponse_ToOptionQuote_Nil(t *testing.T) {
	var resp *quoteResponse
	quote := resp.toOptionQuote()
	if quote != nil {
		t.Error("toOptionQuote() should return nil for nil response")
	}
}

func TestQuoteResponse_ToOptionQuote_Empty(t *testing.T) {
	resp := &quoteResponse{}
	quote := resp.toOptionQuote()
	if quote != nil {
		t.Error("toOptionQuote() should return nil for empty response")
	}
}

func TestQuoteResponse_ToOptionQuotes_NilAndEmpty(t *testing.T) {
	var nilResp *quoteResponse
	if got := nilResp.toOptionQuotes(); got != nil {
		t.Errorf("toOptionQuotes() = %v, want nil for nil response", got)
	}
	if got := (&quoteResponse{}).toOptionQuotes(); got != nil {
		t.Errorf("toOptionQuotes() = %v, want nil for empty response", got)
	}
}

// Test safe index functions
func TestSafeIndex(t *testing.T) {
	slice := []float64{1.0, 2.0, 3.0}
	if safeIndex(slice, 0) != 1.0 {
		t.Error("safeIndex(0) should return 1.0")
	}
	if safeIndex(slice, 5) != 0 {
		t.Error("safeIndex(5) should return 0 for out of bounds")
	}
}

func TestSafeIndexInt(t *testing.T) {
	slice := []int{1, 2, 3}
	if safeIndexInt(slice, 0) != 1 {
		t.Error("safeIndexInt(0) should return 1")
	}
	if safeIndexInt(slice, 5) != 0 {
		t.Error("safeIndexInt(5) should return 0 for out of bounds")
	}
}

func TestSafeIndexInt64(t *testing.T) {
	slice := []int64{1, 2, 3}
	if safeIndexInt64(slice, 0) != 1 {
		t.Error("safeIndexInt64(0) should return 1")
	}
	if safeIndexInt64(slice, 5) != 0 {
		t.Error("safeIndexInt64(5) should return 0 for out of bounds")
	}
}

func TestSafeIndexStr(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if safeIndexStr(slice, 0) != "a" {
		t.Error("safeIndexStr(0) should return 'a'")
	}
	if safeIndexStr(slice, 5) != "" {
		t.Error("safeIndexStr(5) should return empty string for out of bounds")
	}
}

func TestSafeIndexBool(t *testing.T) {
	slice := []bool{true, false, true}
	if !safeIndexBool(slice, 0) {
		t.Error("safeIndexBool(0) should return true")
	}
	if safeIndexBool(slice, 5) {
		t.Error("safeIndexBool(5) should return false for out of bounds")
	}
}

// Test format functions
func TestFormatFloat(t *testing.T) {
	result := formatFloat(150.5)
	if result != "150.5" {
		t.Errorf("formatFloat(150.5) = %q, want 150.5", result)
	}
}

func TestFormatInt(t *testing.T) {
	result := formatInt(30)
	if result != "30" {
		t.Errorf("formatInt(30) = %q, want 30", result)
	}
}

// Test options
func TestWithStrike_StoresFilter(t *testing.T) {
	opts := defaultChainOptions()
	WithStrike(MinStrike(100)).apply(opts)
	if opts.strike == nil {
		t.Fatal("WithStrike should store a StrikeFilter")
	}
	v := url.Values{}
	opts.strike.strike().apply(v)
	if v.Get("strike") != ">=100" {
		t.Errorf("strike = %q, want >=100", v.Get("strike"))
	}
}

func TestWithExpiry_StoresFilter(t *testing.T) {
	opts := defaultChainOptions()
	WithExpiry(InMonth(6)).apply(opts)
	if opts.expiry == nil {
		t.Fatal("WithExpiry should store an ExpiryFilter")
	}
	if opts.expiry.expiry().month != 6 {
		t.Errorf("month = %d, want 6", opts.expiry.expiry().month)
	}
}

func TestWithChainDate_StoresDate(t *testing.T) {
	opts := defaultChainOptions()
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	WithChainDate(date).apply(opts)
	if !opts.date.Equal(date) {
		t.Errorf("date = %v, want %v", opts.date, date)
	}
}

func TestWithOptionQuoteWindow_StoresWindow(t *testing.T) {
	opts := &quoteOptions{}
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	WithOptionQuoteWindow(QuoteOnDate(date)).apply(opts)
	if opts.window == nil {
		t.Fatal("WithOptionQuoteWindow should store an OptionQuoteWindow")
	}
	if !opts.window.optionQuoteWindow().date.Equal(date) {
		t.Errorf("date = %v, want %v", opts.window.optionQuoteWindow().date, date)
	}
}

// Test HTTP error handling
func TestChain_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Chain() should return error for HTTP error")
	}
}

func TestExpirations_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Expirations(context.Background(), "AAPL")
	if err == nil {
		t.Fatal("Expirations() should return error for HTTP error")
	}
}

func TestQuote_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Quote(context.Background(), "AAPL230120C00150000")
	if err == nil {
		t.Fatal("Quote() should return error for HTTP error")
	}
}

func TestLookup_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Lookup(context.Background(), "AAPL", time.Now(), 150.0, Call)
	if err == nil {
		t.Fatal("Lookup() should return error for HTTP error")
	}
}

func TestOptionQuote_String(t *testing.T) {
	c := OptionQuote{OptionSymbol: "AAPL230120C00150000", Strike: 150.0, Type: Call, Bid: 5.0, Ask: 5.5, Last: 5.25, IV: 0.35, Expiration: time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)}
	s := c.String()
	if s == "" {
		t.Error("OptionQuote.String() returned empty string")
	}
}

func TestOptionsChain_String(t *testing.T) {
	oc := OptionsChain{Underlying: "AAPL", Options: make([]OptionQuote, 5)}
	s := oc.String()
	if s == "" {
		t.Error("OptionsChain.String() returned empty string")
	}
}

func TestQuotes_Batch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"TEST"},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.0},
			Ask:          []float64{5.5},
			Last:         []float64{5.25},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	quotes, _, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2", "SYM3"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(quotes) != 3 {
		t.Errorf("len(quotes) = %d, want 3", len(quotes))
	}
}

// quotesBySymbolHandler answers with a quote for every symbol except those in
// noData, which get a 404 — the API's no-data signal.
func quotesBySymbolHandler(t *testing.T, noData map[string]bool) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path is /v1/options/quotes/{symbol}/
		parts := strings.Split(strings.Trim(r.URL.EscapedPath(), "/"), "/")
		symbol := parts[len(parts)-1]

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		if noData[symbol] {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"s": "no_data"})
			return
		}
		_ = json.NewEncoder(w).Encode(quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{symbol},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.0},
			Ask:          []float64{5.5},
			Last:         []float64{5.25},
		})
	})
}

// TestQuotesBySymbol_NoDataIsObservable is the whole reason the method exists:
// a contract the API has no data for must come back as a nil entry under its
// own key, not silently vanish the way Quotes drops it.
func TestQuotesBySymbol_NoDataIsObservable(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, map[string]bool{"SYM2": true}))

	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1", "SYM2", "SYM3"})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("len(quotes) = %d, want 3 (one entry per requested symbol)", len(quotes))
	}
	for _, sym := range []string{"SYM1", "SYM3"} {
		if q, ok := quotes[sym]; !ok || q == nil {
			t.Errorf("quotes[%q] = %v (present=%v), want a quote", sym, q, ok)
		}
	}
	q, ok := quotes["SYM2"]
	if !ok {
		t.Fatal(`quotes["SYM2"] missing: a no-data symbol must still have an entry`)
	}
	if q != nil {
		t.Errorf(`quotes["SYM2"] = %v, want nil for no-data`, q)
	}

	// The contrast with Quotes is the point: it drops the symbol entirely.
	slice, _, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2", "SYM3"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(slice) != 2 {
		t.Errorf("len(Quotes()) = %d, want 2 — the no-data symbol should be omitted", len(slice))
	}
}

// TestQuotesBySymbol_KeysMatchRequestedSymbols pins that entries are keyed by
// the symbol the caller asked for. Keying off the response body instead would
// break exactly when it matters most: a no-data 404 has no body to key from.
func TestQuotesBySymbol_KeysMatchRequestedSymbols(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, map[string]bool{"MISSING": true}))

	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"WANTED", "MISSING"})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	for _, sym := range []string{"WANTED", "MISSING"} {
		if _, ok := quotes[sym]; !ok {
			t.Errorf("quotes[%q] missing; got keys %v", sym, quotes)
		}
	}
}

// TestQuotesBySymbol_Duplicates proves a repeated symbol collapses to one entry
// and that the surviving entry keeps the quote rather than being overwritten
// with nil.
func TestQuotesBySymbol_Duplicates(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, nil))

	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1", "SYM1", "SYM2"})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("len(quotes) = %d, want 2 (duplicates collapse)", len(quotes))
	}
	if quotes["SYM1"] == nil {
		t.Error(`quotes["SYM1"] = nil, want the quote to survive deduplication`)
	}
}

// TestQuotesBySymbol_Single covers the no-concurrency path taken for one symbol.
func TestQuotesBySymbol_Single(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, nil))

	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1"})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	if len(quotes) != 1 || quotes["SYM1"] == nil {
		t.Errorf("quotes = %v, want a single non-nil SYM1 entry", quotes)
	}
}

// TestQuotesBySymbol_SingleNoData covers the one-symbol path when that symbol
// has no data: the entry must exist and be nil.
func TestQuotesBySymbol_SingleNoData(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, map[string]bool{"SYM1": true}))

	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1"})
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	q, ok := quotes["SYM1"]
	if !ok || q != nil {
		t.Errorf("quotes = %v, want a present nil SYM1 entry", quotes)
	}
}

func TestQuotesBySymbol_Empty(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, _, err := svc.QuotesBySymbol(context.Background(), nil)
	if err == nil {
		t.Fatal("QuotesBySymbol() should return error for empty symbols")
	}
	var valErr *sdkerrors.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("error type = %T, want *sdkerrors.ValidationError", err)
	}
}

// TestQuotesBySymbol_Error proves a hard failure is surfaced instead of being
// reported as a map of nils, which would look identical to "no data anywhere".
func TestQuotesBySymbol_Error(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	quotes, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1", "SYM2"})
	if err == nil {
		t.Fatal("QuotesBySymbol() should return the underlying error")
	}
	if quotes != nil {
		t.Errorf("quotes = %v, want nil on error", quotes)
	}
}

func TestGetQuotesBySymbol(t *testing.T) {
	svc := newTestService(quotesBySymbolHandler(t, nil))

	quotes, err := svc.GetQuotesBySymbol([]string{"SYM1", "SYM2"})
	if err != nil {
		t.Fatalf("GetQuotesBySymbol() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Errorf("len(quotes) = %d, want 2", len(quotes))
	}
}

func TestQuotes_Empty(t *testing.T) {
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, _, err := svc.Quotes(context.Background(), nil)
	if err == nil {
		t.Fatal("Quotes() should return error for empty symbols")
	}
	var valErr *sdkerrors.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("error type = %T, want *sdkerrors.ValidationError", err)
	}
}

func TestQuotes_Single(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Api-Ratelimit-Limit", "10000")
		w.Header().Set("X-Api-Ratelimit-Remaining", "9999")
		resp := quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"TEST"},
			Underlying:   []string{"AAPL"},
			Expiration:   []int64{1674172800},
			Strike:       []float64{150.0},
			Side:         []string{"call"},
			Bid:          []float64{5.0},
			Ask:          []float64{5.5},
			Last:         []float64{5.25},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	svc := newTestService(handler)
	quotes, _, err := svc.Quotes(context.Background(), []string{"SYM1"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(quotes) != 1 {
		t.Errorf("len(quotes) = %d, want 1", len(quotes))
	}
}

// --- Convenience method tests ---

func chainHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":              "ok",
			"optionSymbol":   []string{"AAPL230120C00150000"},
			"underlying":     []string{"AAPL"},
			"expiration":     []int64{1674172800},
			"side":           []string{"call"},
			"strike":         []float64{150.0},
			"firstTraded":    []int64{1674000000},
			"dte":            []int{30},
			"ask":            []float64{5.50},
			"askSize":        []int{10},
			"bid":            []float64{5.00},
			"bidSize":        []int{20},
			"mid":            []float64{5.25},
			"last":           []float64{5.15},
			"openInterest":   []int{1000},
			"volume":         []int{500},
			"inTheMoney":     []bool{true},
			"intrinsicValue": []float64{2.0},
			"extrinsicValue": []float64{3.15},
			"updated":        []int64{1704067200},
			"iv":             []float64{0.3},
			"delta":          []float64{0.65},
			"gamma":          []float64{0.02},
			"theta":          []float64{-0.05},
			"vega":           []float64{0.15},
		})
	})
}

func quoteHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":              "ok",
			"optionSymbol":   []string{"AAPL230120C00150000"},
			"ask":            []float64{5.50},
			"askSize":        []int{10},
			"bid":            []float64{5.00},
			"bidSize":        []int{20},
			"mid":            []float64{5.25},
			"last":           []float64{5.15},
			"volume":         []int{500},
			"openInterest":   []int{1000},
			"iv":             []float64{0.3},
			"delta":          []float64{0.65},
			"gamma":          []float64{0.02},
			"theta":          []float64{-0.05},
			"vega":           []float64{0.15},
			"updated":        []int64{1704067200},
			"underlying":     []string{"AAPL"},
			"strike":         []float64{150.0},
			"expiration":     []int64{1674172800},
			"side":           []string{"call"},
			"dte":            []int{30},
			"inTheMoney":     []bool{true},
			"intrinsicValue": []float64{2.0},
			"extrinsicValue": []float64{3.15},
		})
	})
}

func TestGetChain(t *testing.T) {
	svc := newTestService(chainHandler())
	chain, err := svc.GetChain("AAPL")
	if err != nil {
		t.Fatalf("GetChain() error = %v", err)
	}
	if chain == nil {
		t.Fatal("GetChain() returned nil")
	}
	if chain.Underlying != "AAPL" {
		t.Errorf("Underlying = %q, want AAPL", chain.Underlying)
	}
	if len(chain.Options) != 1 {
		t.Fatalf("Options len = %d, want 1", len(chain.Options))
	}
	if chain.Options[0].OptionSymbol != "AAPL230120C00150000" {
		t.Errorf("OptionSymbol = %q, want AAPL230120C00150000", chain.Options[0].OptionSymbol)
	}
}

func TestGetExpirations(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":           "ok",
			"expirations": []int64{1674172800, 1676764800},
		})
	})

	svc := newTestService(handler)
	exps, err := svc.GetExpirations("AAPL")
	if err != nil {
		t.Fatalf("GetExpirations() error = %v", err)
	}
	if exps == nil || len(exps.Dates) != 2 {
		t.Fatalf("Expirations = %v, want 2 dates", exps)
	}
}

func TestGetQuote(t *testing.T) {
	svc := newTestService(quoteHandler())
	quote, err := svc.GetQuote("AAPL230120C00150000")
	if err != nil {
		t.Fatalf("GetQuote() error = %v", err)
	}
	if quote == nil {
		t.Fatal("GetQuote() returned nil")
	}
	if quote.OptionSymbol != "AAPL230120C00150000" {
		t.Errorf("OptionSymbol = %q, want AAPL230120C00150000", quote.OptionSymbol)
	}
	if quote.Bid != 5.00 {
		t.Errorf("Bid = %f, want 5.00", quote.Bid)
	}
}

func TestGetQuotes(t *testing.T) {
	svc := newTestService(quoteHandler())
	quotes, err := svc.GetQuotes([]string{"AAPL230120C00150000", "AAPL230120P00150000"})
	if err != nil {
		t.Fatalf("GetQuotes() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("Quotes len = %d, want 2", len(quotes))
	}
}

func TestGetLookup(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": "AAPL230120C00150000",
		})
	})

	svc := newTestService(handler)
	expDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)
	sym, err := svc.GetLookup("AAPL", expDate, 150.0, Call)
	if err != nil {
		t.Fatalf("GetLookup() error = %v", err)
	}
	if sym != "AAPL230120C00150000" {
		t.Errorf("symbol = %q, want AAPL230120C00150000", sym)
	}
}

// TestQuote_CountbackWindows pins the wire form of the countback quote
// windows. The API pairs countback with `to` and never with `from`, so the
// countback modes must never emit a `from` or a `date`.
func TestQuote_CountbackWindows(t *testing.T) {
	to := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		window OptionQuoteWindow
		want   map[string]string
		absent []string
	}{
		{
			// LastN carries a to= anchor of today: the endpoint ignores a
			// bare countback (verified live). from/date must still be absent.
			name:   "LastN",
			window: QuoteLastN(3),
			want:   map[string]string{"countback": "3", "to": time.Now().In(timezone.Eastern).Format("2006-01-02")},
			absent: []string{"from", "date"},
		},
		{
			name:   "LastNUntil",
			window: QuoteLastNUntil(3, to),
			want:   map[string]string{"countback": "3", "to": "2026-08-08"},
			absent: []string{"from", "date"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				for k, v := range tc.want {
					if got := q.Get(k); got != v {
						t.Errorf("%s = %q, want %q", k, got, v)
					}
				}
				for _, k := range tc.absent {
					if q.Has(k) {
						t.Errorf("%s must not be sent, got %q", k, q.Get(k))
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"s": "ok", "optionSymbol": []string{"AAPL260821C00300000"},
					"underlying": []string{"AAPL"}, "expiration": []int64{1787342400},
					"side": []string{"call"}, "strike": []float64{300},
				})
			})
			svc := newTestService(handler)
			_, _, err := svc.Quote(context.Background(), "AAPL260821C00300000",
				WithOptionQuoteWindow(tc.window))
			if err != nil {
				t.Fatalf("Quote() error = %v", err)
			}
		})
	}
}

// multiRowQuoteHandler answers with n rows, the shape the API returns for a
// historical window (one row per day).
func multiRowQuoteHandler(rows int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		syms := make([]string, rows)
		bids := make([]float64, rows)
		for i := range syms {
			syms[i] = "AAPL260821C00300000"
			bids[i] = float64(i + 1)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "optionSymbol": syms, "bid": bids,
		})
	})
}

// TestQuoteHistory_ReturnsEveryRow is the reason QuoteHistory exists: a
// countback or a range makes the API return one row per day, and Quote keeps
// only the first. Asking for three quotes must yield three.
func TestQuoteHistory_ReturnsEveryRow(t *testing.T) {
	svc := newTestService(multiRowQuoteHandler(3))

	quotes, _, err := svc.QuoteHistory(context.Background(), "AAPL260821C00300000",
		WithOptionQuoteWindow(QuoteLastN(3)))
	if err != nil {
		t.Fatalf("QuoteHistory() error = %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("len(quotes) = %d, want 3 — rows are being dropped", len(quotes))
	}
	// Order must be the API's, not reversed or sorted.
	for i, q := range quotes {
		if q.Bid != float64(i+1) {
			t.Errorf("quotes[%d].Bid = %v, want %v — order is not preserved", i, q.Bid, i+1)
		}
	}

	// The contrast with Quote is the point.
	single, _, err := svc.Quote(context.Background(), "AAPL260821C00300000",
		WithOptionQuoteWindow(QuoteLastN(3)))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if single == nil || single.Bid != 1 {
		t.Errorf("Quote() = %v, want the first row only", single)
	}
}

func TestQuoteHistory_NoData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "no_data"})
	})
	svc := newTestService(handler)

	quotes, resp, err := svc.QuoteHistory(context.Background(), "AAPL260821C00300000")
	if err != nil {
		t.Fatalf("QuoteHistory() error = %v", err)
	}
	if quotes != nil {
		t.Errorf("quotes = %v, want nil on no-data", quotes)
	}
	if resp == nil || !resp.NoData {
		t.Error("want a response with NoData set")
	}
}

func TestQuoteHistory_EmptySymbol(t *testing.T) {
	svc := NewService(nil)
	_, _, err := svc.QuoteHistory(context.Background(), "")
	var valErr *sdkerrors.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("error = %v (%T), want *sdkerrors.ValidationError", err, err)
	}
}

func TestQuoteHistory_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error"})
	})
	svc := newTestService(handler)

	if _, _, err := svc.QuoteHistory(context.Background(), "AAPL260821C00300000"); err == nil {
		t.Error("QuoteHistory() should return an error for a non-ok API status")
	}
}

func TestQuoteHistory_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	svc := newTestService(handler)

	if _, _, err := svc.QuoteHistory(context.Background(), "AAPL260821C00300000"); err == nil {
		t.Error("QuoteHistory() should return the underlying error")
	}
}

func TestGetQuoteHistory(t *testing.T) {
	svc := newTestService(multiRowQuoteHandler(2))

	quotes, err := svc.GetQuoteHistory("AAPL260821C00300000", WithOptionQuoteWindow(QuoteLastN(2)))
	if err != nil {
		t.Fatalf("GetQuoteHistory() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Errorf("len(quotes) = %d, want 2", len(quotes))
	}
}

// TestQuote_RejectsBadCountback proves a non-positive countback is caught
// before the request, like every other window rule.
func TestQuote_RejectsBadCountback(t *testing.T) {
	requestMade := false
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))

	_, _, err := svc.Quote(context.Background(), "AAPL260821C00300000",
		WithOptionQuoteWindow(QuoteLastN(0)))
	var valErr *sdkerrors.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("error = %v (%T), want *sdkerrors.ValidationError", err, err)
	}
	if requestMade {
		t.Error("a request reached the server; validation must happen pre-network")
	}
}

// --- Chain option parameter tests ---

// TestChain_AllExpirations pins the wire form of the full-chain request. The
// API returns only the front-month expiration when the expiration parameter is
// absent, so "expiration=all" is the only way to reach the complete chain and
// must actually reach the wire — verified live 2026-08-11: AAPL unfiltered
// returns 190 contracts across 1 expiration, with expiration=all 3528 across 24.
func TestChain_AllExpirations(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("expiration"); got != "all" {
			t.Errorf("expiration = %q, want all", got)
		}
		// The whole point of "all" is that no other expiry selector is sent
		// alongside it: dte or month/year would narrow the chain back down.
		for _, p := range []string{"dte", "month", "year", "from", "to"} {
			if r.URL.Query().Has(p) {
				t.Errorf("%s must not be sent with expiration=all, got %q", p, r.URL.Query().Get(p))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpiry(AllExpirations()))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

// TestChain_NoExpiryFilterSendsNoExpiration pins the counterpart contract: an
// unfiltered chain must not invent an expiration parameter. It is the
// front-month request the API documents, and the distinction from
// AllExpirations is the entire reason that constructor exists.
func TestChain_NoExpiryFilterSendsNoExpiration(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("expiration") {
			t.Errorf("expiration must be absent without an expiry filter, got %q", r.URL.Query().Get("expiration"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_ExpirationBetween(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "2024-01-01" {
			t.Errorf("from = %q, want 2024-01-01", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "2024-06-30" {
			t.Errorf("to = %q, want 2024-06-30", r.URL.Query().Get("to"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpiry(ExpirationBetween(fromDate, toDate)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithExpirationTypes(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Exclude weeklies and quarterlies -> weekly=false&quarterly=false.
		if r.URL.Query().Get("quarterly") != "false" {
			t.Errorf("quarterly = %q, want false", r.URL.Query().Get("quarterly"))
		}
		if r.URL.Query().Get("weekly") != "false" {
			t.Errorf("weekly = %q, want false", r.URL.Query().Get("weekly"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithExpirationTypes(ExcludeExpirationTypes(Weekly, Quarterly)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithMaxBid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("maxBid") != "10.5" {
			t.Errorf("maxBid = %q, want 10.5", r.URL.Query().Get("maxBid"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithMaxBid(10.5))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithMinAsk(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("minAsk") != "1.5" {
			t.Errorf("minAsk = %q, want 1.5", r.URL.Query().Get("minAsk"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithMinAsk(1.5))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithMaxAsk(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("maxAsk") != "20" {
			t.Errorf("maxAsk = %q, want 20", r.URL.Query().Get("maxAsk"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithMaxAsk(20))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithMaxBidAskSpreadPct(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("maxBidAskSpreadPct") != "0.05" {
			t.Errorf("maxBidAskSpreadPct = %q, want 0.05", r.URL.Query().Get("maxBidAskSpreadPct"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithMaxBidAskSpreadPct(0.05))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithAM(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("am") != "true" {
			t.Errorf("am = %q, want true", r.URL.Query().Get("am"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithAM(true))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestChain_WithPM(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pm") != "true" {
			t.Errorf("pm = %q, want true", r.URL.Query().Get("pm"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{},
			"underlying":   []string{},
			"expiration":   []int64{},
			"side":         []string{},
			"strike":       []float64{},
		})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithPM(true))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestQuote_WithOptionQuoteRange(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("from") != "2024-01-01" {
			t.Errorf("from = %q, want 2024-01-01", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "2024-06-30" {
			t.Errorf("to = %q, want 2024-06-30", r.URL.Query().Get("to"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s":            "ok",
			"optionSymbol": []string{"AAPL230120C00150000"},
			"underlying":   []string{"AAPL"},
			"expiration":   []int64{1674172800},
			"strike":       []float64{150.0},
			"side":         []string{"call"},
			"bid":          []float64{5.0},
			"ask":          []float64{5.5},
			"last":         []float64{5.25},
		})
	})

	svc := newTestService(handler)
	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, _, err := svc.Quote(context.Background(), "AAPL230120C00150000",
		WithOptionQuoteWindow(QuoteRange(fromDate, toDate)))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
}

// --- 404 tests ---

func TestChain_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		// The API's real markerless no-data body (verified live
		// 2026-09-01): a valid question whose answer is empty. An
		// errmsg here would mean the question itself was rejected,
		// which is a different case — see the *_404NotFound tests.
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	})

	svc := newTestService(handler)
	chain, resp, err := svc.Chain(context.Background(), "NOSYMBOL")
	if err != nil {
		t.Fatalf("Chain() error = %v, want nil for 404", err)
	}
	// Non-nil and empty, not nil: the query was valid and matched nothing,
	// so the answer is the empty set. A nil pointer here panics the caller
	// who ranges the result, which is what happened in the PR #33 CI run.
	if chain == nil {
		t.Fatal("Chain() returned a nil chain for a markerless 404; want an empty one")
	}
	if len(chain.Options) != 0 {
		t.Errorf("Chain() options = %d, want 0", len(chain.Options))
	}
	for range chain.Options { // must not panic
		t.Error("unreachable")
	}
	if resp == nil {
		t.Fatal("Chain() should return response for 404")
	}
	if !resp.NoData {
		t.Error("Chain() response NoData = false, want true")
	}
}

func TestExpirations_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		// The API's real markerless no-data body (verified live
		// 2026-09-01): a valid question whose answer is empty. An
		// errmsg here would mean the question itself was rejected,
		// which is a different case — see the *_404NotFound tests.
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	})

	svc := newTestService(handler)
	exps, resp, err := svc.Expirations(context.Background(), "NOSYMBOL")
	if err != nil {
		t.Fatalf("Expirations() error = %v, want nil for 404", err)
	}
	if exps == nil {
		t.Fatal("Expirations() returned nil for a markerless 404; want an empty result")
	}
	if len(exps.Dates) != 0 {
		t.Errorf("Expirations() dates = %d, want 0", len(exps.Dates))
	}
	if resp == nil {
		t.Fatal("Expirations() should return response for 404")
	}
	if !resp.NoData {
		t.Error("Expirations() response NoData = false, want true")
	}
}

func TestQuote_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		// The API's real markerless no-data body (verified live
		// 2026-09-01): a valid question whose answer is empty. An
		// errmsg here would mean the question itself was rejected,
		// which is a different case — see the *_404NotFound tests.
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	})

	svc := newTestService(handler)
	quote, resp, err := svc.Quote(context.Background(), "BADSYMBOL")
	if err != nil {
		t.Fatalf("Quote() error = %v, want nil for 404", err)
	}
	// Still nil, deliberately: OptionQuote is scalar-shaped, so a
	// zero-valued struct would read as a real quote priced at zero — a
	// silently wrong answer, strictly worse than a nil the caller must
	// check. Only the collection-shaped types gained an empty value.
	if quote != nil {
		t.Error("Quote() should return nil quote for 404")
	}
	if resp == nil {
		t.Fatal("Quote() should return response for 404")
	}
	if !resp.NoData {
		t.Error("Quote() response NoData = false, want true")
	}
}

func TestLookup_404(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		// The API's real markerless no-data body (verified live
		// 2026-09-01): a valid question whose answer is empty. An
		// errmsg here would mean the question itself was rejected,
		// which is a different case — see the *_404NotFound tests.
		_, _ = w.Write([]byte(`{"s":"no_data","nextTime":null,"prevTime":null}`))
	})

	svc := newTestService(handler)
	sym, resp, err := svc.Lookup(context.Background(), "BADSYM", time.Now(), 150.0, Call)
	if err != nil {
		t.Fatalf("Lookup() error = %v, want nil for 404", err)
	}
	if sym != "" {
		t.Errorf("Lookup() symbol = %q, want empty for 404", sym)
	}
	if resp == nil {
		t.Fatal("Lookup() should return response for 404")
	}
}

// --- Error path tests ---

func TestQuotes_WithError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2"})
	if err == nil {
		t.Fatal("Quotes() should return error when a symbol fetch fails")
	}
}

func TestChain_MinStrikeOnly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		strike := r.URL.Query().Get("strike")
		if strike != ">=140" {
			t.Errorf("strike = %q, want >=140", strike)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{Status: "ok", OptionSymbol: []string{}})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithStrike(MinStrike(140)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestQuotes_MultipleWithNilQuote(t *testing.T) {
	// Multi-symbol Quotes where one returns 404 (nil quote)
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	quotes, _, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2"})
	if err != nil {
		t.Fatalf("Quotes() should not error, got: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("len(quotes) = %d, want 0 (all 404)", len(quotes))
	}
}

func TestChain_MaxStrikeOnly(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		strike := r.URL.Query().Get("strike")
		if strike != "<=160" {
			t.Errorf("strike = %q, want <=160", strike)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{Status: "ok", OptionSymbol: []string{}})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL", WithStrike(MaxStrike(160)))
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

func TestQuotes_MultipleSuccess(t *testing.T) {
	// Multi-symbol Quotes where all return valid data
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{
			Status:       "ok",
			OptionSymbol: []string{"SYM"},
			Ask:          []float64{5.5},
			Bid:          []float64{5.0},
			Last:         []float64{5.25},
			Updated:      []int64{1704067200},
		})
	}))
	quotes, resp, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2"})
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}
	if len(quotes) != 2 {
		t.Errorf("len(quotes) = %d, want 2", len(quotes))
	}
	if resp == nil {
		t.Error("resp should not be nil")
	}
}

func TestChain_PathInjection(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quoteResponse{Status: "ok", OptionSymbol: []string{}})
	})

	svc := newTestService(handler)
	_, _, err := svc.Chain(context.Background(), "AAPL/../../user")
	if err != nil {
		t.Fatalf("Chain() error = %v", err)
	}

	want := "/v1/options/chain/AAPL%2F..%2F..%2Fuser/"
	if gotPath != want {
		t.Errorf("request path = %q, want %q (symbol must not escape its path segment)", gotPath, want)
	}
}

func TestQuotes_SingleWithError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	svc := newTestService(handler)
	_, _, err := svc.Quotes(context.Background(), []string{"SYM1"})
	if err == nil {
		t.Fatal("Quotes() should return error when the single-symbol fetch fails")
	}
}

func TestQuotes_SingleNoData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	svc := newTestService(handler)
	quotes, resp, err := svc.Quotes(context.Background(), []string{"SYM1"})
	if err != nil {
		t.Fatalf("Quotes() error = %v, want nil for 404 no-data", err)
	}
	if quotes != nil {
		t.Errorf("quotes = %v, want nil", quotes)
	}
	if resp == nil || !resp.NoData {
		t.Error("resp.NoData = false, want true for single-symbol 404")
	}
}

// TestQuotes_FirstErrorCancelsSiblings pins ADR-014's cancellation: once a
// batch symbol fails hard, in-flight sibling fetches are canceled instead
// of burning API credits, and the ROOT error (not a cancellation echo)
// surfaces to the caller.
//
// Regression note (T-2): the sibling handler used to block on
// `<-release`/`<-r.Context().Done()` with no bound, and `release` only
// closes via the deferred close() at the end of THIS function — which
// never runs if svc.Quotes never returns. So if cancellation regressed,
// this test didn't fail fast with a readable message; it deadlocked until
// the package-level `go test` timeout (10 minutes by default), which is
// how CI would have reported the bug. The 3s fallback below, paired with
// the elapsed assertion well under it, makes a missing cancellation fail
// in ~3s with a clear message instead.
func TestQuotes_FirstErrorCancelsSiblings(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "BAD") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"s":"error","errmsg":"boom"}`))
			return
		}
		// Siblings block until canceled by the batch, or the bounded
		// fallback fires.
		select {
		case <-release:
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok"}`))
	}))
	defer server.Close()

	client := internalhttp.New(internalhttp.Config{
		BaseURL: server.URL, APIVersion: "v1", Token: "k",
		RetryCfg:   retry.Config{MaxRetries: 0, InitialBackoff: time.Millisecond, Multiplier: 2},
		RateLimits: ratelimit.New(),
	})
	svc := NewService(client)

	start := time.Now()
	_, _, err := svc.Quotes(context.Background(), []string{"AAPLBAD", "AAPL260918C00230000", "AAPL260918P00230000"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Quotes() should surface the failing symbol's error")
	}
	if errors.Is(err, context.Canceled) {
		t.Errorf("Quotes() returned a cancellation echo (%v); want the root 400 error", err)
	}
	// Without cancellation, siblings block until the 3s fallback fires; with
	// it, the batch returns promptly after the first failure.
	if elapsed > 1*time.Second {
		t.Errorf("Quotes() took %v; sibling fetches were not canceled", elapsed)
	}
}

// TestQuotes_AppliesWindowToEverySymbol pins a reverse Java parity fix: the batch methods now
// take the same QuoteOption values as Quote and apply them to every
// symbol in the fan-out. Before this, a historical window was expressible for
// one contract but not for several.
func TestQuotes_AppliesWindowToEverySymbol(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{} // symbol -> date parameter

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.EscapedPath(), "/"), "/")
		symbol := parts[len(parts)-1]
		mu.Lock()
		seen[symbol] = r.URL.Query().Get("date")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "optionSymbol": []string{symbol}, "bid": []float64{1},
		})
	})

	svc := newTestService(handler)
	day := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.Quotes(context.Background(), []string{"SYM1", "SYM2", "SYM3"},
		WithOptionQuoteWindow(QuoteOnDate(day)))
	if err != nil {
		t.Fatalf("Quotes() error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 3 {
		t.Fatalf("saw %d symbols, want 3", len(seen))
	}
	for sym, got := range seen {
		if got != "2026-08-08" {
			t.Errorf("%s: date = %q, want 2026-08-08 — the window is not reaching every symbol", sym, got)
		}
	}
}

// TestQuotesBySymbol_AppliesWindow is the map-shaped counterpart.
func TestQuotesBySymbol_AppliesWindow(t *testing.T) {
	var mu sync.Mutex
	var counts int

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if r.URL.Query().Get("countback") == "3" {
			counts++
		}
		mu.Unlock()
		parts := strings.Split(strings.Trim(r.URL.EscapedPath(), "/"), "/")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"s": "ok", "optionSymbol": []string{parts[len(parts)-1]}, "bid": []float64{1},
		})
	})

	svc := newTestService(handler)
	to := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	_, _, err := svc.QuotesBySymbol(context.Background(), []string{"SYM1", "SYM2"},
		WithOptionQuoteWindow(QuoteLastNUntil(3, to)))
	if err != nil {
		t.Fatalf("QuotesBySymbol() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if counts != 2 {
		t.Errorf("countback reached %d of 2 symbols", counts)
	}
}

// TestLookupQuery pins a reverse Java parity fix: the endpoint takes a free-form human description,
// which the typed Lookup cannot express. The whole query travels as one escaped
// path segment.
func TestLookupQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slashes inside the query are percent-encoded rather than left raw:
		// the whole description is one path segment, and PathSegment
		// neutralizes separators so a crafted query cannot re-route the
		// request. The endpoint documents that the input arrives
		// URL-encoded.
		want := "/v1/options/lookup/AAPL%207%2F26%2F23%20$200%20Call/"
		if got := r.URL.EscapedPath(); got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok", "optionSymbol": "AAPL230726C00200000"})
	})

	svc := newTestService(handler)
	sym, _, err := svc.LookupQuery(context.Background(), "AAPL 7/26/23 $200 Call")
	if err != nil {
		t.Fatalf("LookupQuery() error = %v", err)
	}
	if sym != "AAPL230726C00200000" {
		t.Errorf("symbol = %q, want AAPL230726C00200000", sym)
	}
}

func TestLookupQuery_RejectsBlank(t *testing.T) {
	requestMade := false
	svc := newTestService(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))

	for _, q := range []string{"", "   ", "\t\n"} {
		_, _, err := svc.LookupQuery(context.Background(), q)
		var valErr *sdkerrors.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("LookupQuery(%q) error = %v (%T), want *ValidationError", q, err, err)
		}
	}
	if requestMade {
		t.Error("a blank query reached the server; validation must happen pre-network")
	}
}

func TestLookupQuery_NoData(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	svc := newTestService(handler)

	sym, resp, err := svc.LookupQuery(context.Background(), "NOPE 1/1/99 $1 Call")
	if err != nil {
		t.Fatalf("LookupQuery() error = %v", err)
	}
	if sym != "" || resp == nil || !resp.NoData {
		t.Errorf("got (%q, %v), want an empty symbol and a NoData response", sym, resp)
	}
}

func TestLookupQuery_BadStatus(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "error"})
	})
	svc := newTestService(handler)

	if _, _, err := svc.LookupQuery(context.Background(), "AAPL 7/26/23 $200 Call"); err == nil {
		t.Error("LookupQuery() should error on a non-ok API status")
	}
}

func TestLookupQuery_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	svc := newTestService(handler)

	if _, _, err := svc.LookupQuery(context.Background(), "AAPL 7/26/23 $200 Call"); err == nil {
		t.Error("LookupQuery() should return the underlying error")
	}
}

func TestGetLookupQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"s": "ok", "optionSymbol": "AAPL230726C00200000"})
	})
	svc := newTestService(handler)

	sym, err := svc.GetLookupQuery("AAPL 7/26/23 $200 Call")
	if err != nil || sym != "AAPL230726C00200000" {
		t.Errorf("GetLookupQuery() = (%q, %v)", sym, err)
	}
}

// TestChain_MoneynessIsTyped pins a reverse Java parity fix: the moneyness filter is a closed set
// of API keywords rather than a free string, so a typo cannot reach the wire as
// a silently-ignored filter.
func TestChain_MoneynessIsTyped(t *testing.T) {
	cases := []struct {
		name string
		m    Moneyness
		want string
	}{
		{"ITM", MoneynessITM, "itm"},
		{"OTM", MoneynessOTM, "otm"},
		{"All", MoneynessAll, "all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("range"); got != tc.want {
					t.Errorf("range = %q, want %q", got, tc.want)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "optionSymbol": []string{}})
			})
			svc := newTestService(handler)
			if _, _, err := svc.Chain(context.Background(), "AAPL", WithRange(tc.m)); err != nil {
				t.Fatalf("Chain() error = %v", err)
			}
		})
	}

	// MoneynessUnset leaves the parameter off entirely.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("range") {
			t.Errorf("range = %q, want the parameter absent", r.URL.Query().Get("range"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"s": "ok", "optionSymbol": []string{}})
	})
	svc := newTestService(handler)
	if _, _, err := svc.Chain(context.Background(), "AAPL", WithRange(MoneynessUnset)); err != nil {
		t.Fatalf("Chain() error = %v", err)
	}
}

// TestOptionQuote_DecodesRho pins a reverse Java parity fix. The API models rho internally but does
// not serialize it today, so the field is normally zero; this asserts the
// decode path is wired, using hand-written wire JSON in the shape the API would
// send.
func TestOptionQuote_DecodesRho(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","optionSymbol":["AAPL260821C00300000"],` +
			`"delta":[0.55],"gamma":[0.01],"theta":[-0.05],"vega":[0.15],"rho":[0.0812]}`))
	})

	svc := newTestService(handler)
	q, _, err := svc.Quote(context.Background(), "AAPL260821C00300000")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if q.Rho != 0.0812 {
		t.Errorf("Rho = %v, want 0.0812", q.Rho)
	}

	// Absent rho (today's real shape) decodes to zero rather than failing.
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s":"ok","optionSymbol":["AAPL260821C00300000"],"vega":[0.15]}`))
	})
	q2, _, err := newTestService(handler2).Quote(context.Background(), "AAPL260821C00300000")
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if q2.Rho != 0 {
		t.Errorf("Rho = %v, want 0 when the API omits it", q2.Rho)
	}
}
