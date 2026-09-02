package stocks

import (
	"strings"
	"testing"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

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
	if !containsString(errStr, "symbol") {
		t.Errorf("Error() should contain field name: %s", errStr)
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

func TestQuoteNotFoundError_Error(t *testing.T) {
	err := &QuoteNotFoundError{
		Symbol: "INVALID",
	}
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
	if !containsString(errStr, "INVALID") {
		t.Errorf("Error() should contain symbol: %s", errStr)
	}
}

// Test safe index functions
func TestSafeIndex_EdgeCases(t *testing.T) {
	// Empty slice
	if safeIndex([]float64{}, 0) != 0 {
		t.Error("safeIndex on empty slice should return 0")
	}

	// Nil equivalent
	if safeIndex(nil, 0) != 0 {
		t.Error("safeIndex on nil should return 0")
	}

	// Valid access
	if safeIndex([]float64{1.5, 2.5, 3.5}, 1) != 2.5 {
		t.Error("safeIndex[1] should return 2.5")
	}
}

func TestSafeIndexInt_EdgeCases(t *testing.T) {
	if safeIndexInt([]int{}, 0) != 0 {
		t.Error("safeIndexInt on empty slice should return 0")
	}
	if safeIndexInt(nil, 0) != 0 {
		t.Error("safeIndexInt on nil should return 0")
	}
	if safeIndexInt([]int{1, 2, 3}, 2) != 3 {
		t.Error("safeIndexInt[2] should return 3")
	}
}

func TestSafeIndexInt64_EdgeCases(t *testing.T) {
	if safeIndexInt64([]int64{}, 0) != 0 {
		t.Error("safeIndexInt64 on empty slice should return 0")
	}
	if safeIndexInt64(nil, 0) != 0 {
		t.Error("safeIndexInt64 on nil should return 0")
	}
	if safeIndexInt64([]int64{1, 2, 3}, 2) != 3 {
		t.Error("safeIndexInt64[2] should return 3")
	}
}

func TestSafeIndexString_EdgeCases(t *testing.T) {
	if safeIndexString([]string{}, 0) != "" {
		t.Error("safeIndexString on empty slice should return empty string")
	}
	if safeIndexString(nil, 0) != "" {
		t.Error("safeIndexString on nil should return empty string")
	}
	if safeIndexString([]string{"a", "b", "c"}, 1) != "b" {
		t.Error("safeIndexString[1] should return 'b'")
	}
}

// Test response conversion methods
func TestQuotesResponse_ToQuotes_Nil(t *testing.T) {
	var resp *quotesResponse
	quotes := resp.toQuotes()
	if quotes != nil {
		t.Error("toQuotes() should return nil for nil response")
	}
}

func TestQuotesResponse_ToQuotes_Empty(t *testing.T) {
	resp := &quotesResponse{}
	quotes := resp.toQuotes()
	if quotes != nil {
		t.Error("toQuotes() should return nil for empty response")
	}
}

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

func TestPricesResponse_ToPrices_Nil(t *testing.T) {
	var resp *pricesResponse
	prices := resp.toPrices()
	if prices != nil {
		t.Error("toPrices() should return nil for nil response")
	}
}

func TestPricesResponse_ToPrices_Empty(t *testing.T) {
	resp := &pricesResponse{}
	prices := resp.toPrices()
	if prices != nil {
		t.Error("toPrices() should return nil for empty response")
	}
}

func TestEarningsResponse_ToEarnings_Nil(t *testing.T) {
	var resp *earningsResponse
	earnings := resp.toEarnings()
	if earnings != nil {
		t.Error("toEarnings() should return nil for nil response")
	}
}

func TestEarningsResponse_ToEarnings_Empty(t *testing.T) {
	resp := &earningsResponse{}
	earnings := resp.toEarnings()
	if earnings != nil {
		t.Error("toEarnings() should return nil for empty response")
	}
}

func TestNewsResponse_ToNewsArticles_Nil(t *testing.T) {
	var resp *newsResponse
	articles := resp.toNewsArticles()
	if articles != nil {
		t.Error("toNewsArticles() should return nil for nil response")
	}
}

func TestNewsResponse_ToNewsArticles_Empty(t *testing.T) {
	resp := &newsResponse{}
	articles := resp.toNewsArticles()
	if articles != nil {
		t.Error("toNewsArticles() should return nil for empty response")
	}
}

func TestBulkCandlesResponse_ToBulkCandles_Nil(t *testing.T) {
	var resp *bulkCandlesResponse
	candles := resp.toBulkCandles(nil)
	if candles != nil {
		t.Error("toBulkCandles() should return nil for nil response")
	}
}

func TestBulkCandlesResponse_ToBulkCandles_Empty(t *testing.T) {
	resp := &bulkCandlesResponse{}
	candles := resp.toBulkCandles(nil)
	if candles != nil {
		t.Error("toBulkCandles() should return nil for empty response")
	}
}

// TestBulkCandlesResponse_ToBulkCandles_NoSymbolArray covers the live
// single-symbol shape, where the API sends candle data and no symbol array.
// The row count must come from t, and the symbol is restored from the
// caller's request.
func TestBulkCandlesResponse_ToBulkCandles_NoSymbolArray(t *testing.T) {
	resp := &bulkCandlesResponse{
		Status: "ok",
		Time:   []int64{1704067200},
		Open:   []float64{170.11}, High: []float64{171.22},
		Low: []float64{169.33}, Close: []float64{170.99},
		Volume: []int64{45000111},
	}

	candles := resp.toBulkCandles([]string{"AAPL"})
	if len(candles) != 1 {
		t.Fatalf("len = %d, want 1 — the candle must not be dropped", len(candles))
	}
	if candles[0].Symbol != "AAPL" || candles[0].Close != 170.99 {
		t.Errorf("candle = %+v, want AAPL with close 170.99", candles[0])
	}

	// With no requested symbol to fall back on, the data still survives; only
	// the label is empty.
	bare := resp.toBulkCandles(nil)
	if len(bare) != 1 || bare[0].Close != 170.99 {
		t.Errorf("toBulkCandles(nil) = %+v, want the candle with an empty symbol", bare)
	}
	if bare[0].Symbol != "" {
		t.Errorf("Symbol = %q, want empty when neither wire nor request supplies one", bare[0].Symbol)
	}

	// More than one row is ambiguous, so the fallback deliberately does not
	// fire: a wrong label is worse than a missing one.
	two := &bulkCandlesResponse{Status: "ok", Time: []int64{1, 2}, Close: []float64{1, 2}}
	got := two.toBulkCandles([]string{"AAPL"})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Symbol != "" || got[1].Symbol != "" {
		t.Errorf("symbols = %q/%q, want both empty", got[0].Symbol, got[1].Symbol)
	}
}

// Test defaultCandleOptions
func TestDefaultCandleOptions(t *testing.T) {
	opts := defaultCandleOptions()
	if opts.resolution != ResolutionDaily {
		t.Errorf("default resolution = %q, want D", opts.resolution)
	}
}

// fp builds a *float64 wire element for earnings test literals.
func fp(v float64) *float64 { return &v }

// Test earnings response with nullable fields
func TestEarningsResponse_ToEarnings_WithNullableFields(t *testing.T) {
	resp := &earningsResponse{
		Symbol:         []string{"AAPL"},
		FiscalYear:     []int{2024},
		FiscalQuarter:  []int{1},
		Date:           []int64{1704067200},
		ReportDate:     []int64{1704153600},
		ReportTime:     []string{"after close"},
		Currency:       []string{"USD"},
		ReportedEPS:    []*float64{fp(2.18)},
		EstimatedEPS:   []*float64{fp(2.10)},
		SurpriseEPS:    []*float64{fp(0.08)},
		SurpriseEPSPct: []*float64{fp(3.81)},
		Updated:        []int64{1704067200},
	}

	earnings := resp.toEarnings()
	if len(earnings) != 1 {
		t.Fatalf("toEarnings() len = %d, want 1", len(earnings))
	}

	e := earnings[0]
	if e.ReportedEPS == nil {
		t.Error("ReportedEPS should not be nil")
	} else if *e.ReportedEPS != 2.18 {
		t.Errorf("ReportedEPS = %f, want 2.18", *e.ReportedEPS)
	}

	if e.EstimatedEPS == nil {
		t.Error("EstimatedEPS should not be nil")
	}
	if e.SurpriseEPS == nil {
		t.Error("SurpriseEPS should not be nil")
	}
	if e.SurpriseEPSPercent == nil {
		t.Error("SurpriseEPSPercent should not be nil")
	}
}

// Test that a true $0.00 EPS survives as a pointer to zero while a wire
// null (nil element) maps to nil — the distinction the Earning type
// promises in the package documentation.
func TestEarningsResponse_ToEarnings_ZeroEPSIsReal(t *testing.T) {
	resp := &earningsResponse{
		Symbol:         []string{"AAPL"},
		FiscalYear:     []int{2024},
		FiscalQuarter:  []int{1},
		Date:           []int64{1704067200},
		ReportDate:     []int64{1704153600},
		ReportTime:     []string{},
		Currency:       []string{},
		ReportedEPS:    []*float64{fp(0)}, // met a $0.00 estimate exactly
		EstimatedEPS:   []*float64{fp(0)},
		SurpriseEPS:    []*float64{fp(0)}, // met expectations
		SurpriseEPSPct: []*float64{nil},   // wire null
		Updated:        []int64{1704067200},
	}

	earnings := resp.toEarnings()
	e := earnings[0]

	if e.ReportedEPS == nil || *e.ReportedEPS != 0 {
		t.Errorf("ReportedEPS = %v, want pointer to 0 for a real $0.00", e.ReportedEPS)
	}
	if e.EstimatedEPS == nil || *e.EstimatedEPS != 0 {
		t.Errorf("EstimatedEPS = %v, want pointer to 0 for a real $0.00", e.EstimatedEPS)
	}
	if e.SurpriseEPS == nil || *e.SurpriseEPS != 0 {
		t.Errorf("SurpriseEPS = %v, want pointer to 0 when expectations were met", e.SurpriseEPS)
	}
	if e.SurpriseEPSPercent != nil {
		t.Errorf("SurpriseEPSPercent = %v, want nil for a wire null", e.SurpriseEPSPercent)
	}

	// A met-expectations quarter must render as $0.00, not n/a.
	if s := e.String(); !strings.Contains(s, "Surprise: $0.00") {
		t.Errorf("String() = %q, want it to contain %q", s, "Surprise: $0.00")
	}
}

// Test candle methods
func TestCandle_IsDoji(t *testing.T) {
	// Doji: open == close
	doji := Candle{Open: 100, Close: 100, High: 105, Low: 95}
	if doji.IsBullish() || doji.IsBearish() {
		t.Error("Doji candle should be neither bullish nor bearish")
	}
}

// Test full quotes conversion with all fields
func TestQuotesResponse_ToQuotes_Full(t *testing.T) {
	resp := &quotesResponse{
		Status:    "ok",
		Symbol:    []string{"AAPL", "MSFT"},
		Ask:       []float64{150.25, 375.50},
		AskSize:   []int{100, 50},
		Bid:       []float64{150.20, 375.45},
		BidSize:   []int{200, 100},
		Mid:       []float64{150.225, 375.475},
		Last:      []float64{150.22, 375.48},
		Change:    []float64{1.50, 2.25},
		ChangePct: []float64{1.01, 0.60},
		Volume:    []int64{50000000, 25000000},
		Updated:   []int64{1704067200, 1704067200},
	}

	quotes := resp.toQuotes()
	if len(quotes) != 2 {
		t.Fatalf("toQuotes() len = %d, want 2", len(quotes))
	}

	q := quotes[0]
	if q.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", q.Symbol)
	}
	if q.Ask != 150.25 {
		t.Errorf("Ask = %f, want 150.25", q.Ask)
	}
	if q.AskSize != 100 {
		t.Errorf("AskSize = %d, want 100", q.AskSize)
	}
	if q.BidSize != 200 {
		t.Errorf("BidSize = %d, want 200", q.BidSize)
	}
}

// Test String() methods
func TestQuote_String(t *testing.T) {
	q := Quote{Symbol: "AAPL", Last: 150.22, Change: 1.50, ChangePercent: 1.01, Volume: 50000000}
	s := q.String()
	if !strings.Contains(s, "AAPL") || !strings.Contains(s, "150.22") {
		t.Errorf("Quote.String() = %q, want to contain symbol and price", s)
	}
}

// The API sends changepct as a fraction (-0.0021 = -0.21%); String must
// render it ×100 so a typical sub-percent day doesn't display as 0.00%.
func TestQuote_String_RendersFractionAsPercent(t *testing.T) {
	q := Quote{Symbol: "AAPL", ChangePercent: -0.0021}
	if s := q.String(); !strings.Contains(s, "(-0.21%)") {
		t.Errorf("Quote.String() = %q, want it to contain %q", s, "(-0.21%)")
	}
}

func TestPrice_String_RendersFractionAsPercent(t *testing.T) {
	p := Price{Symbol: "AAPL", ChangePercent: 0.0067}
	if s := p.String(); !strings.Contains(s, "(0.67%)") {
		t.Errorf("Price.String() = %q, want it to contain %q", s, "(0.67%)")
	}
}

func TestEarning_String_RendersSurprisePctAsPercent(t *testing.T) {
	pct := 0.0256
	e := Earning{Symbol: "AAPL", FiscalQuarter: 1, FiscalYear: 2024, SurpriseEPSPercent: &pct}
	if s := e.String(); !strings.Contains(s, "(2.56%)") {
		t.Errorf("Earning.String() = %q, want it to contain %q (a percentage, not $-formatted)", s, "(2.56%)")
	}
}

func TestCandle_String(t *testing.T) {
	c := Candle{Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 95, Close: 102, Volume: 1000}
	s := c.String()
	if !strings.Contains(s, "2024-01-15") || !strings.Contains(s, "100.00") {
		t.Errorf("Candle.String() = %q, want to contain date and price", s)
	}
}

func TestPrice_String(t *testing.T) {
	p := Price{Symbol: "AAPL", Mid: 150.22, Change: 1.50, ChangePercent: 1.01}
	s := p.String()
	if !strings.Contains(s, "AAPL") || !strings.Contains(s, "150.22") {
		t.Errorf("Price.String() = %q, want to contain symbol and price", s)
	}
}

func TestEarning_String(t *testing.T) {
	eps := 2.18
	e := Earning{Symbol: "AAPL", FiscalQuarter: 1, FiscalYear: 2024, ReportedEPS: &eps, ReportDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC)}
	s := e.String()
	if !strings.Contains(s, "AAPL") || !strings.Contains(s, "Q1") || !strings.Contains(s, "2.18") {
		t.Errorf("Earning.String() = %q, want to contain symbol, quarter, and EPS", s)
	}
}

func TestEarning_String_NoEPS(t *testing.T) {
	e := Earning{Symbol: "AAPL", FiscalQuarter: 2, FiscalYear: 2024, ReportDate: time.Date(2024, 4, 25, 0, 0, 0, 0, time.UTC)}
	s := e.String()
	if !strings.Contains(s, "n/a") {
		t.Errorf("Earning.String() = %q, want to contain 'n/a' for nil EPS", s)
	}
}

func TestNewsArticle_String(t *testing.T) {
	n := NewsArticle{Symbol: "AAPL", Headline: "Apple reports earnings", PublicationDate: time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC)}
	s := n.String()
	if !strings.Contains(s, "AAPL") || !strings.Contains(s, "Apple reports earnings") {
		t.Errorf("NewsArticle.String() = %q, want to contain symbol and headline", s)
	}
}

func TestBulkCandle_String(t *testing.T) {
	bc := BulkCandle{Symbol: "AAPL", Time: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 95, Close: 102, Volume: 1000}
	s := bc.String()
	if !strings.Contains(s, "AAPL") || !strings.Contains(s, "2024-01-15") {
		t.Errorf("BulkCandle.String() = %q, want to contain symbol and date", s)
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestQuoteString_FiftyTwoWeek(t *testing.T) {
	q := Quote{
		Symbol:           "AAPL",
		Last:             150.0,
		FiftyTwoWeekHigh: 199.5,
		FiftyTwoWeekLow:  101.25,
	}
	s := q.String()
	if !strings.Contains(s, "52wk: 101.25-199.50") {
		t.Errorf("String() = %q, want 52wk range included", s)
	}
}
