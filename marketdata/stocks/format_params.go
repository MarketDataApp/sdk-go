package stocks

import (
	"net/url"
	"strings"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/params"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
	"github.com/MarketDataApp/sdk-go/v2/internal/timezone"
)

// Request path and query parameter construction for every stocks endpoint.
// These are the single serializer: the JSON methods on [Service] and both
// formatted facets — CSV (exported, [Service.AsCSV]) and HTML (unexported,
// [Service.asHTML]) — all build their request here, so the three can only
// ever differ in wire format and response type, never in what they ask the
// API for. See ADR-018 for the facets; the previously duplicated
// serializers were collapsed because a facet could silently drop an option
// the JSON method sent. There is one deliberate exception, dateformat on
// options expirations, which serves a decoder rather than the request.

func quotePath(symbol string, opts []QuoteOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}
	options := &quoteOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}
	p := url.Values{}
	p.Set("symbols", symbol)
	if options.fiftyTwoWeek {
		p.Set("52week", "true")
	}
	if options.extended != nil {
		p.Set("extended", boolParam(*options.extended))
	}
	if options.candle != nil {
		p.Set("candle", boolParam(*options.candle))
	}
	return "stocks/bulkquotes/", p, nil
}

func quotesPath(symbols []string, opts []QuotesOption) (string, url.Values, error) {
	if len(symbols) == 0 {
		return "", nil, &sdkerrors.ValidationError{Field: "symbols", Message: "at least one symbol is required"}
	}
	options := &quotesOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}
	p := url.Values{}
	p.Set("symbols", strings.Join(symbols, ","))
	if options.extended != nil {
		p.Set("extended", boolParam(*options.extended))
	}
	if options.candle != nil {
		p.Set("candle", boolParam(*options.candle))
	}
	return "stocks/bulkquotes/", p, nil
}

func pricesPath(symbols []string, opts []PriceOption) (string, url.Values, error) {
	if len(symbols) == 0 {
		return "", nil, &sdkerrors.ValidationError{Field: "symbols", Message: "at least one symbol is required"}
	}
	options := &priceOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}
	p := url.Values{}
	var path string
	if len(symbols) == 1 {
		path = "stocks/prices/" + http.PathSegment(symbols[0]) + "/"
	} else {
		path = "stocks/prices/"
		p.Set("symbols", strings.Join(symbols, ","))
	}
	if options.extended != nil {
		p.Set("extended", boolParam(*options.extended))
	}
	return path, p, nil
}

func newsPath(symbol string, opts []NewsOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}
	options := &newsOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}
	if err := options.window.Validate(); err != nil {
		return "", nil, err
	}
	p := url.Values{}
	options.window.Apply(p)
	return "stocks/news/" + http.PathSegment(symbol) + "/", p, nil
}

func earningsPath(symbol string, opts []EarningsOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}
	options := &earningsOptions{}
	for _, opt := range opts {
		opt.apply(options)
	}
	if err := options.window.Validate(); err != nil {
		return "", nil, err
	}
	// Anchor a bare countback to today (Eastern), matching Service.Earnings:
	// without to= the API ignores countback and serves its upcoming-only
	// default window instead of the last n reports.
	window := options.window.AnchorCountback(time.Now().In(timezone.Eastern))
	p := url.Values{}
	window.Apply(p)
	if options.report != "" {
		p.Set("report", options.report)
	}
	return "stocks/earnings/" + http.PathSegment(symbol) + "/", p, nil
}

// candlesPath validates the symbol and options and returns the candles path
// plus the chunk params to fetch — one element for a single request, or
// several for a range split into disjoint year-sized chunks.
//
// It is the single plan behind all three candle facets: [Service.Candles],
// [CSVService.Candles] and the HTML facet each consume it and differ only in
// what they do with the chunks. Service.Candles used to duplicate the
// validation and the split decision instead, bound to this copy by nothing
// but a comment.
func candlesPath(symbol string, opts []CandleOption) (string, []url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}
	options := defaultCandleOptions()
	for _, opt := range opts {
		opt.apply(options)
	}
	if err := options.window.Validate(); err != nil {
		return "", nil, err
	}

	path := candlesEndpoint(symbol, options.resolution)

	if isIntraday(options.resolution) && options.window.IsRange() &&
		options.window.To().Sub(options.window.From()) > 365*24*time.Hour {
		// Each chunk is built from the whole option set with only the
		// window narrowed, never field by field: a field-by-field copy
		// silently drops any option added later, which is how a chunked
		// request for a non-US listing once came back as the US one.
		chunks := candleChunks(options.window)
		chunkParams := make([]url.Values, len(chunks))
		for i, c := range chunks {
			chunkParams[i] = candleParams(options, options.window.Chunk(c.from, c.to))
		}
		return path, chunkParams, nil
	}

	return path, []url.Values{candleParams(options, options.window)}, nil
}

// isIntraday returns true for intraday resolutions.
func isIntraday(r Resolution) bool {
	switch r {
	case Resolution1Min, Resolution3Min, Resolution5Min, Resolution15Min, Resolution30Min, Resolution45Min, Resolution1Hour, Resolution2Hour, Resolution4Hour:
		return true
	}
	return false
}

// candlesEndpoint builds the candles path. The resolution is a path segment
// and nothing else: it used to also travel as a query parameter, which the
// API ignores entirely — verified live, a request to /candles/D/ with
// resolution=60 still returns daily candles.
func candlesEndpoint(symbol string, resolution Resolution) string {
	return "stocks/candles/" + http.PathSegment(resolution.String()) + "/" + http.PathSegment(symbol) + "/"
}

func candleParams(options *candleOptions, window params.Window) url.Values {
	p := url.Values{}
	window.Apply(p)
	if options.extended != nil {
		p.Set("extended", boolParam(*options.extended))
	}
	if options.adjustSplits != nil {
		p.Set("adjustsplits", boolParam(*options.adjustSplits))
	}
	if options.adjustDividends != nil {
		p.Set("adjustdividends", boolParam(*options.adjustDividends))
	}
	return p
}

func bulkCandlesPath(symbols []string, opts []BulkCandleOption) (string, url.Values, error) {
	options := &bulkCandleOptions{resolution: ResolutionDaily}
	for _, opt := range opts {
		opt.apply(options)
	}

	// An empty symbol list is meaningful only as the market-wide snapshot
	// request; without snapshot=true the endpoint has nothing to fetch.
	snapshot := options.snapshot != nil && *options.snapshot
	if len(symbols) == 0 && !snapshot {
		return "", nil, &sdkerrors.ValidationError{
			Field:   "symbols",
			Message: "at least one symbol is required, unless requesting the market-wide snapshot with WithSnapshot(true)",
		}
	}

	p := url.Values{}
	if len(symbols) > 0 {
		p.Set("symbols", strings.Join(symbols, ","))
	}
	if !options.date.IsZero() {
		p.Set("date", options.date.Format("2006-01-02"))
	}
	if options.adjustSplits != nil {
		p.Set("adjustsplits", boolParam(*options.adjustSplits))
	}
	if options.adjustDividends != nil {
		p.Set("adjustdividends", boolParam(*options.adjustDividends))
	}
	if options.snapshot != nil {
		p.Set("snapshot", boolParam(*options.snapshot))
	}

	return "stocks/bulkcandles/" + http.PathSegment(options.resolution.String()) + "/", p, nil
}

func boolParam(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
