package stocks

import (
	"context"
	"net/url"
	"sort"
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/fanout"
	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/response"
)

// Service provides access to the Market Data stocks endpoints: quotes,
// candles, bulk candles, prices, earnings, and news. It is created by the
// parent marketdata client and exposed as its Stocks field; most callers
// never construct a Service directly.
type Service struct {
	http *http.Client
}

// NewService creates a new stocks service that issues requests through the
// given HTTP client. It is called by the parent marketdata client during
// initialization.
func NewService(httpClient *http.Client) *Service {
	return &Service{
		http: httpClient,
	}
}

// Quote fetches a real-time quote for a single stock symbol.
//
// The symbol is required; an empty symbol returns a validation error
// without making a request. Optional behavior is controlled by
// [QuoteOption] values: [WithFiftyTwoWeek] adds 52-week high/low data to
// the quote, and [WithExtended] controls whether extended-hours data is
// included.
//
// A symbol the API does not recognize is reported as a [NotFoundError]. If
// the symbol exists but the API has no data for it, Quote returns a nil
// quote, a nil error, and a non-nil [response.Response] whose NoData field
// is true — a Quote has no meaningful empty value, so callers must check
// for nil. If the API reports success but the response contains no quote
// for the symbol, Quote returns a [QuoteNotFoundError]. See the "Missing
// Data and Unknown Symbols" section of the marketdata package
// documentation.
//
// Quote is served by the bulk quotes endpoint. API documentation:
// https://www.marketdata.app/docs/api/stocks/bulkquotes
//
// Example:
//
//	quote, _, err := client.Stocks.Quote(ctx, "AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if quote != nil {
//	    fmt.Printf("AAPL last: $%.2f\n", quote.Last)
//	}
func (s *Service) Quote(ctx context.Context, symbol string, opts ...QuoteOption) (*Quote, *response.Response, error) {
	path, params, err := quotePath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp quotesResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	quotes := apiResp.toQuotes()
	if len(quotes) == 0 {
		return nil, nil, &QuoteNotFoundError{Symbol: symbol}
	}
	return &quotes[0], response.New(httpResp), nil
}

// Quotes fetches real-time quotes for multiple stock symbols in a single
// API request.
//
// At least one symbol is required; an empty slice returns a validation
// error without making a request. [WithQuotesExtended] controls whether
// extended-hours data is included.
//
// The bulk quotes endpoint does not provide 52-week high/low data (the API
// ignores the 52week parameter), so [WithFiftyTwoWeek] is not accepted
// here; use [Service.Quote] per symbol when 52-week data is needed.
//
// If the API responds with 404 because no data is available, Quotes
// returns a nil slice, a nil error, and a non-nil [response.Response]
// whose NoData field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/bulkquotes
//
// Example:
//
//	quotes, _, err := client.Stocks.Quotes(ctx, []string{"AAPL", "MSFT", "GOOG"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, q := range quotes {
//	    fmt.Printf("%s: $%.2f\n", q.Symbol, q.Last)
//	}
func (s *Service) Quotes(ctx context.Context, symbols []string, opts ...QuotesOption) ([]Quote, *response.Response, error) {
	path, params, err := quotesPath(symbols, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp quotesResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toQuotes(), response.New(httpResp), nil
}

// Candles fetches historical OHLCV candles for a stock symbol.
//
// The symbol is required; an empty symbol returns a validation error
// without making a request. The resolution defaults to [ResolutionDaily]
// and is changed with [WithResolution]. The date range is set with
// [WithCandleWindow], which takes a [DateWindow] built by a helper such as
// [Between] (an explicit from/to range), [Since] or [Until] (an open-ended
// range), [OnDate] (a single day), or [LastN]/[LastNUntil] (a fixed number
// of the most recent candles instead of a range). [WithCandleExtended]
// controls extended-hours data and [WithCandleAdjustSplits] controls
// adjustment for stock splits. Window dates are sent to the API as calendar
// dates, so any time-of-day component is ignored.
//
// For intraday resolutions ([Resolution1Min] through [Resolution4Hour]),
// when the window is a bounded range that spans more than one year,
// Candles automatically splits the range into disjoint year-sized chunks
// (consecutive chunks never share a boundary day, so no candle is fetched
// twice), fetches the chunks concurrently, and merges the results in
// chronological order. In that case the returned [response.Response]
// reflects the last underlying request that carried data, and its NoData
// field is true only when the whole range has no data.
//
// If the API responds with 404 because no data is available, Candles
// returns a nil slice, a nil error, and a non-nil [response.Response]
// whose NoData field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/candles
//
// Example:
//
//	candles, _, err := client.Stocks.Candles(ctx, "AAPL",
//	    stocks.WithResolution(stocks.ResolutionDaily),
//	    stocks.WithCandleWindow(
//	        stocks.Between(time.Now().AddDate(0, -1, 0), time.Now())),
//	)
func (s *Service) Candles(ctx context.Context, symbol string, opts ...CandleOption) ([]Candle, *response.Response, error) {
	// Validation, the endpoint path, and the chunk-split decision all come
	// from candlesPath, the same builder the CSV and HTML facets consume.
	// This method used to carry its own copy of all three; the copies were
	// bound only by a comment and had already drifted apart textually, which
	// is the failure shape the shared builder closed for the other eleven
	// JSON methods.
	// One plan, three consumers: a chunk list of one is a single request.
	path, chunkParams, err := candlesPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}
	if len(chunkParams) == 1 {
		return s.candlesSingle(ctx, path, chunkParams[0])
	}
	return s.candlesSplit(ctx, path, chunkParams)
}

// candlesSingle fetches candles for a single date range.
func (s *Service) candlesSingle(ctx context.Context, path string, query url.Values) ([]Candle, *response.Response, error) {
	var apiResp candlesResponse
	httpResp, err := s.http.Get(ctx, path, query, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toCandles(), response.New(httpResp), nil
}

// candlesSplit splits a large date range into year-sized chunks, fetches them
// concurrently (respecting the global concurrency pool), and merges the results.
func (s *Service) candlesSplit(ctx context.Context, path string, chunkParams []url.Values) ([]Candle, *response.Response, error) {
	// Fetch all chunks concurrently, abandoning the rest on the first
	// failure (ADR-014, see fanout.Run): once the merge is going to fail
	// there is no reason to keep spending API credits on the remaining
	// chunks.
	type chunkResult struct {
		candles []Candle
		resp    *response.Response
	}
	results, err := fanout.Run(ctx, len(chunkParams), func(ctx context.Context, i int) (chunkResult, error) {
		candles, resp, err := s.candlesSingle(ctx, path, chunkParams[i])
		return chunkResult{candles: candles, resp: resp}, err
	})
	if err != nil {
		return nil, nil, err
	}

	// Merge results. The returned Response is the last one that carried
	// data, so a trailing NoData chunk (e.g. a range ending in days with no
	// trades) cannot mark a result that does have candles as NoData; only
	// if every chunk is NoData does the NoData response win.
	var allCandles []Candle
	var lastResp *response.Response
	var lastDataResp *response.Response
	for _, r := range results {
		allCandles = append(allCandles, r.candles...)
		if r.resp != nil {
			lastResp = r.resp
			if !r.resp.NoData {
				lastDataResp = r.resp
			}
		}
	}
	if lastDataResp != nil {
		lastResp = lastDataResp
	}

	// Sort by time to ensure chronological order
	sort.Slice(allCandles, func(i, j int) bool {
		return allCandles[i].Time.Before(allCandles[j].Time)
	})

	// Defense in depth against boundary overlap: a candle timestamp is unique
	// per symbol and resolution, so identical adjacent timestamps after the
	// sort can only be the same candle fetched by two chunks.
	deduped := allCandles[:0]
	for i, c := range allCandles {
		if i > 0 && c.Time.Equal(allCandles[i-1].Time) {
			continue
		}
		deduped = append(deduped, c)
	}

	return deduped, lastResp, nil
}

// afterDate reports whether a's calendar date (Year/Month/Day, in a's own
// location) is after b's calendar date (in b's own location) — the
// comparison implied by window.Apply's YYYY-MM-DD wire serialization,
// regardless of time-of-day or whether a and b are in different zones.
func afterDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	if ay != by {
		return ay > by
	}
	if am != bm {
		return am > bm
	}
	return ad > bd
}

// Prices fetches real-time SmartMid midpoint prices for one or more stock
// symbols. The SmartMid price is Market Data's derived midpoint, a
// lightweight alternative to a full quote when only a single price per
// symbol is needed.
//
// At least one symbol is required; an empty slice returns a validation
// error without making a request. The optional [WithPriceExtended] option
// controls whether extended-hours data is included.
//
// If the API responds with 404 because no data is available, Prices
// returns a nil slice, a nil error, and a non-nil [response.Response]
// whose NoData field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/prices
//
// Example:
//
//	prices, _, err := client.Stocks.Prices(ctx, []string{"AAPL", "MSFT", "GOOG"})
func (s *Service) Prices(ctx context.Context, symbols []string, opts ...PriceOption) ([]Price, *response.Response, error) {
	path, params, err := pricesPath(symbols, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp pricesResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toPrices(), response.New(httpResp), nil
}

// Earnings fetches earnings reports for a stock symbol, one [Earning] per
// fiscal quarter.
//
// The symbol is required; an empty symbol returns a validation error
// without making a request. With no options the API returns its default
// set of reports (the upcoming scheduled ones). [WithEarningsWindow]
// bounds the results by date; it takes a [DateWindow] built by a helper
// such as [Between], [Since], or [Until] (a date range), [OnDate] (the
// report covering a single date), or [LastN]/[LastNUntil] (a fixed number
// of the most recent reports). Dates are sent to the API as calendar
// dates, so any time-of-day component is ignored.
//
// [LastN] is sent with an explicit to= anchor of today (Eastern): the
// earnings endpoint silently ignores a countback that arrives without
// to, degrading the request to the upcoming-only default window. The
// anchor pins the promised "n most recent reports" semantics and remains
// correct (and harmless) if the API-side behavior is fixed.
//
// EPS fields on the returned [Earning] values are pointers and are nil
// when the API reports no value, such as for earnings not yet reported.
//
// If the API responds with 404 because no data is available, Earnings
// returns a nil slice, a nil error, and a non-nil [response.Response]
// whose NoData field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/earnings
//
// Example:
//
//	earnings, _, err := client.Stocks.Earnings(ctx, "AAPL")
func (s *Service) Earnings(ctx context.Context, symbol string, opts ...EarningsOption) ([]Earning, *response.Response, error) {
	path, params, err := earningsPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp earningsResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toEarnings(), response.New(httpResp), nil
}

// News fetches news articles for a stock symbol.
//
// The symbol is required; an empty symbol returns a validation error
// without making a request. With no options the API returns its default
// set of articles. [WithNewsWindow] bounds the results by date; it takes a
// [DateWindow] built by a helper such as [Between], [Since], or [Until] (a
// date range), [OnDate] (articles from a single date), or [LastN] /
// [LastNUntil] (a fixed number of the most recent articles). Dates are sent
// to the API as calendar dates, so any time-of-day component is ignored.
//
// If the API responds with 404 because no data is available, News returns
// a nil slice, a nil error, and a non-nil [response.Response] whose NoData
// field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/news
//
// Example:
//
//	news, _, err := client.Stocks.News(ctx, "AAPL")
func (s *Service) News(ctx context.Context, symbol string, opts ...NewsOption) ([]NewsArticle, *response.Response, error) {
	path, params, err := newsPath(symbol, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp newsResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toNewsArticles(), response.New(httpResp), nil
}

// BulkCandles fetches daily candles for multiple stock symbols in a single
// API request.
//
// At least one symbol is required, except in the market-wide snapshot form
// described below; otherwise an empty slice returns a validation error
// without making a request. The resolution defaults to
// [ResolutionDaily], the only resolution the bulk candles endpoint
// supports. [WithBulkDate] requests candles for a specific historical
// date, [WithAdjustSplits] controls adjustment for stock splits, and
// [WithSnapshot] controls whether the API returns a snapshot of the latest
// candle.
//
// Passing no symbols together with WithSnapshot(true) requests the
// market-wide snapshot: the API returns a candle for every symbol it covers,
// which is many thousands of rows and is billed accordingly. It is the only
// way to obtain that snapshot, and the only case in which an empty symbol
// list is accepted.
//
// If the API responds with 404 because no data is available, BulkCandles
// returns a nil slice, a nil error, and a non-nil [response.Response]
// whose NoData field is true.
//
// API documentation: https://www.marketdata.app/docs/api/stocks/bulkcandles
//
// Example:
//
//	candles, _, err := client.Stocks.BulkCandles(ctx, []string{"AAPL", "MSFT", "GOOG"})
//
//	// Every symbol the API covers — expensive:
//	all, _, err := client.Stocks.BulkCandles(ctx, nil, stocks.WithSnapshot(true))
func (s *Service) BulkCandles(ctx context.Context, symbols []string, opts ...BulkCandleOption) ([]BulkCandle, *response.Response, error) {
	path, params, err := bulkCandlesPath(symbols, opts)
	if err != nil {
		return nil, nil, err
	}

	var apiResp bulkCandlesResponse
	httpResp, err := s.http.Get(ctx, path, params, &apiResp)
	if err != nil {
		return nil, nil, err
	}

	if response.IsNoData(httpResp.StatusCode) {
		return nil, response.NewNoData(httpResp), nil
	}

	if apiResp.Status != "ok" {
		return nil, nil, httpResp.StatusError(apiResp.Status)
	}

	return apiResp.toBulkCandles(symbols), response.New(httpResp), nil
}

// --- Convenience methods (no context, no *Response) ---
// These use context.Background() and return only (data, error).

// GetQuote is a convenience wrapper for [Service.Quote] that uses
// context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil quote with a nil error.
func (s *Service) GetQuote(symbol string, opts ...QuoteOption) (*Quote, error) {
	q, _, err := s.Quote(context.Background(), symbol, opts...)
	return q, err
}

// GetQuotes is a convenience wrapper for [Service.Quotes] that uses
// context.Background, accepts symbols variadically, and discards the
// per-request [response.Response]. It does not accept options; use
// [Service.Quotes] when a [QuoteOption] is needed. Because the response
// metadata is discarded, a 404 no-data result is returned as a nil slice
// with a nil error.
func (s *Service) GetQuotes(symbols ...string) ([]Quote, error) {
	q, _, err := s.Quotes(context.Background(), symbols)
	return q, err
}

// GetCandles is a convenience wrapper for [Service.Candles] that uses
// context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil slice with a nil error.
func (s *Service) GetCandles(symbol string, opts ...CandleOption) ([]Candle, error) {
	c, _, err := s.Candles(context.Background(), symbol, opts...)
	return c, err
}

// GetPrices is a convenience wrapper for [Service.Prices] that uses
// context.Background, accepts symbols variadically, and discards the
// per-request [response.Response]. It does not accept options; use
// [Service.Prices] when a [PriceOption] is needed. Because the response
// metadata is discarded, a 404 no-data result is returned as a nil slice
// with a nil error.
func (s *Service) GetPrices(symbols ...string) ([]Price, error) {
	p, _, err := s.Prices(context.Background(), symbols)
	return p, err
}

// GetBulkCandles is a convenience wrapper for [Service.BulkCandles] that uses
// context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil slice with a nil error.
//
// It takes the symbol slice and options in the same shape as BulkCandles,
// including the market-wide snapshot form, which passes no symbols:
//
//	all, err := client.Stocks.GetBulkCandles(nil, stocks.WithSnapshot(true))
func (s *Service) GetBulkCandles(symbols []string, opts ...BulkCandleOption) ([]BulkCandle, error) {
	b, _, err := s.BulkCandles(context.Background(), symbols, opts...)
	return b, err
}

// GetEarnings is a convenience wrapper for [Service.Earnings] that uses
// context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil slice with a nil error.
func (s *Service) GetEarnings(symbol string, opts ...EarningsOption) ([]Earning, error) {
	e, _, err := s.Earnings(context.Background(), symbol, opts...)
	return e, err
}

// GetNews is a convenience wrapper for [Service.News] that uses
// context.Background and discards the per-request [response.Response].
// Because the response metadata is discarded, a 404 no-data result is
// returned as a nil slice with a nil error.
func (s *Service) GetNews(symbol string, opts ...NewsOption) ([]NewsArticle, error) {
	n, _, err := s.News(context.Background(), symbol, opts...)
	return n, err
}
