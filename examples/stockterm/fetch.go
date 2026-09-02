// fetch.go is the reference implementation for calling the Market Data Go
// SDK from a Bubble Tea program. Every fetch function follows the same
// four-step shape:
//
//  1. Derive a bounded context from context.Background() with
//     context.WithTimeout(ctx, fetchTimeout), and defer its cancel. Bubble
//     Tea commands run outside any request-scoped context, so each fetch
//     owns its own deadline.
//  2. Call exactly one SDK service method, passing whatever functional
//     options the endpoint needs. Nothing else touches the client.
//  3. On error, return errMsg{op, err} unchanged — do not classify or
//     unwrap it here. format.go's classify (task 2.3) and the Update loop
//     (task 2.4) do that with errors.As against the SDK's typed errors
//     (*marketdata.RateLimitError, *marketdata.AuthenticationError,
//     *marketdata.NetworkError, ...).
//  4. On success, return the operation's typed message carrying the
//     decoded data and the per-request *marketdata.Response (conventionally
//     named meta), which holds rate-limit state.
//
// "Success" includes the SDK's 404-as-no-data convention: when the API
// reports no data for a valid request, every context-first SDK method
// returns a nil error and a *marketdata.Response with NoData set to true,
// not an error. Fetch functions pass that straight through as their normal
// typed message (nil/empty data, meta.NoData == true) rather than routing
// it through errMsg — a missing quote for a delisted symbol is not a
// failure of the fetch itself. validateSymbol goes one step further: the
// SDK also reports "no quote for this symbol" as a *stocks.QuoteNotFoundError
// when the API answered 200 but the result set was empty, and validateSymbol
// treats that the same as a 404 (noData: true), since from the caller's
// perspective both mean "the API answered; there's no quote."
//
// errMsg op strings, one per fetch function: "quotes", "prices",
// "candles", "fund-candles", "quote-52wk", "earnings", "news",
// "market-status", "status-history", "user", "api-status", "headers",
// "bulk-candles", "validate".
package main

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/funds"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/markets"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/stocks"
)

// fetchTimeout bounds every SDK call made by the app.
const fetchTimeout = 15 * time.Second

// fetchQuotes loads real-time quotes for the watchlist via
// client.Stocks.Quotes. No options are used; it is the default watchlist
// refresh, replaced by fetchPrices under -prices.
func fetchQuotes(client *marketdata.Client, symbols []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		quotes, resp, err := client.Stocks.Quotes(ctx, symbols)
		if err != nil {
			return errMsg{op: "quotes", err: err}
		}
		return quotesMsg{quotes: quotes, meta: resp}
	}
}

// fetchPrices loads lightweight SmartMid prices for the watchlist via
// client.Stocks.Prices. No options are used; it is the -prices alternative
// to fetchQuotes.
func fetchPrices(client *marketdata.Client, symbols []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		prices, resp, err := client.Stocks.Prices(ctx, symbols)
		if err != nil {
			return errMsg{op: "prices", err: err}
		}
		return pricesMsg{prices: prices, meta: resp}
	}
}

// fetchCandles loads the detail pane's candle window for symbol via
// client.Stocks.Candles. The options depend on rng: rangeIntraday uses
// stocks.WithResolution(stocks.Resolution5Min) with
// stocks.WithCandleWindow(stocks.LastN(78)) (one trading day of 5-minute
// bars); rangeWeekly uses stocks.WithResolution(stocks.ResolutionWeekly)
// with stocks.WithCandleWindow(stocks.LastN(52)) (one year of weekly
// bars); rangeDaily (the default) uses
// stocks.WithResolution(stocks.ResolutionDaily) with
// stocks.WithCandleWindow(stocks.Between(...)) spanning the trailing 3
// months.
func fetchCandles(client *marketdata.Client, symbol string, rng candleRange) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		var opts []stocks.CandleOption
		switch rng {
		case rangeIntraday:
			opts = []stocks.CandleOption{
				stocks.WithResolution(stocks.Resolution5Min),
				stocks.WithCandleWindow(stocks.LastN(78)),
			}
		case rangeWeekly:
			opts = []stocks.CandleOption{
				stocks.WithResolution(stocks.ResolutionWeekly),
				stocks.WithCandleWindow(stocks.LastN(52)),
			}
		default: // rangeDaily
			now := time.Now()
			opts = []stocks.CandleOption{
				stocks.WithResolution(stocks.ResolutionDaily),
				stocks.WithCandleWindow(stocks.Between(now.AddDate(0, -3, 0), now)),
			}
		}

		candles, resp, err := client.Stocks.Candles(ctx, symbol, opts...)
		if err != nil {
			return errMsg{op: "candles", err: err}
		}
		return candlesMsg{symbol: symbol, rng: rng, candles: candles, meta: resp}
	}
}

// fetchFundCandles loads the detail pane's candle window for a mutual
// fund symbol via client.Funds.Candles, used instead of fetchCandles when
// the symbol is in the model's funds set. It requests roughly 3 months of
// daily NAV candles: funds.WithResolution(funds.ResolutionDaily) with
// funds.WithCandleWindow(funds.LastN(63)) (about 63 trading days).
func fetchFundCandles(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		candles, resp, err := client.Funds.Candles(ctx, symbol,
			funds.WithResolution(funds.ResolutionDaily),
			funds.WithCandleWindow(funds.LastN(63)),
		)
		if err != nil {
			return errMsg{op: "fund-candles", err: err}
		}
		return fundCandlesMsg{symbol: symbol, candles: candles, meta: resp}
	}
}

// fetchDetailQuote loads the 52-week high/low for the selected symbol via
// client.Stocks.Quote with stocks.WithFiftyTwoWeek(true). Unlike
// validateSymbol, a *stocks.QuoteNotFoundError here is treated as an
// ordinary error (errMsg): by the time a symbol reaches the detail pane
// it has already been validated onto the watchlist, so a missing quote
// signals a real problem rather than an expected "invalid ticker" outcome.
func fetchDetailQuote(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		quote, resp, err := client.Stocks.Quote(ctx, symbol, stocks.WithFiftyTwoWeek(true))
		if err != nil {
			return errMsg{op: "quote-52wk", err: err}
		}
		return detailQuoteMsg{symbol: symbol, quote: quote, meta: resp}
	}
}

// fetchEarnings loads upcoming earnings reports for the selected symbol
// via client.Stocks.Earnings, bounded to a 180-day forward window with
// stocks.WithEarningsWindow(stocks.Between(today, today+180d)).
func fetchEarnings(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		today := time.Now()
		earnings, resp, err := client.Stocks.Earnings(ctx, symbol,
			stocks.WithEarningsWindow(stocks.Between(today, today.AddDate(0, 0, 180))),
		)
		if err != nil {
			return errMsg{op: "earnings", err: err}
		}
		return earningsMsg{symbol: symbol, earnings: earnings, meta: resp}
	}
}

// fetchNews loads the 3 most recent news articles for the selected symbol
// via client.Stocks.News with stocks.WithNewsWindow(stocks.LastN(3)).
func fetchNews(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		articles, resp, err := client.Stocks.News(ctx, symbol, stocks.WithNewsWindow(stocks.LastN(3)))
		if err != nil {
			return errMsg{op: "news", err: err}
		}
		return newsMsg{symbol: symbol, articles: articles, meta: resp}
	}
}

// fetchMarketStatus loads today's US market status via
// client.Markets.Status. No options are used; it is refreshed on the 60s
// market tick.
func fetchMarketStatus(client *marketdata.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		status, resp, err := client.Markets.Status(ctx)
		if err != nil {
			return errMsg{op: "market-status", err: err}
		}
		return marketStatusMsg{status: status, meta: resp}
	}
}

// fetchStatusHistory loads the 5 most recent days of US market status via
// client.Markets.StatusHistory, using
// markets.WithHistoryWindow(markets.LastNUntil(5, now)). Shown in the
// status-history modal.
func fetchStatusHistory(client *marketdata.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		statuses, resp, err := client.Markets.StatusHistory(ctx,
			markets.WithHistoryWindow(markets.LastNUntil(5, time.Now())),
		)
		if err != nil {
			return errMsg{op: "status-history", err: err}
		}
		return statusHistoryMsg{statuses: statuses, meta: resp}
	}
}

// fetchUser loads the authenticated account's credit state via
// client.Utilities.User. No options are used.
func fetchUser(client *marketdata.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		user, resp, err := client.Utilities.User(ctx)
		if err != nil {
			return errMsg{op: "user", err: err}
		}
		return userMsg{user: user, meta: resp}
	}
}

// fetchAPIStatus loads the Market Data API's own uptime status via
// client.Utilities.Status. No options are used; shown in the diagnostics
// modal.
func fetchAPIStatus(client *marketdata.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		status, resp, err := client.Utilities.Status(ctx)
		if err != nil {
			return errMsg{op: "api-status", err: err}
		}
		return apiStatusMsg{status: status, meta: resp}
	}
}

// fetchHeaders loads the request headers the client actually sent via
// client.Utilities.Headers. No options are used; shown in the diagnostics
// modal to help debug authentication issues.
func fetchHeaders(client *marketdata.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		headers, resp, err := client.Utilities.Headers(ctx)
		if err != nil {
			return errMsg{op: "headers", err: err}
		}
		return headersMsg{headers: headers, meta: resp}
	}
}

// fetchBulkCandles loads today's daily candle for every watchlist symbol
// in one request via client.Stocks.BulkCandles. No options are used, so
// the endpoint's default resolution (daily) applies. Shown in the
// day-performance modal.
func fetchBulkCandles(client *marketdata.Client, symbols []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		candles, resp, err := client.Stocks.BulkCandles(ctx, symbols)
		if err != nil {
			return errMsg{op: "bulk-candles", err: err}
		}
		return bulkCandlesMsg{candles: candles, meta: resp}
	}
}

// validateSymbol checks whether symbol has a quote before it is added to
// the watchlist, via client.Stocks.Quote. No options are used. Two
// distinct "no quote" signals from the SDK are both reported as
// addValidatedMsg{noData: true} rather than errMsg, since either way the
// API answered and simply has nothing for this symbol:
//
//   - An ordinary 404: err is nil, resp.NoData is true.
//   - A *stocks.QuoteNotFoundError: the API responded 200 but the result
//     set was empty (a shape unique to the single-quote endpoint).
//
// Any other error is a genuine failure and is returned as errMsg.
func validateSymbol(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		quote, resp, err := client.Stocks.Quote(ctx, symbol)
		if err != nil {
			var notFound *stocks.QuoteNotFoundError
			if errors.As(err, &notFound) {
				// resp is nil here by SDK contract: Stocks.Quote returns
				// (nil, nil, *QuoteNotFoundError) on the 200-but-no-quote
				// path, so this message carries meta == nil. That nil is
				// part of addValidatedMsg's documented contract — see the
				// type's doc comment in app.go.
				return addValidatedMsg{symbol: symbol, noData: true, meta: resp}
			}
			return errMsg{op: "validate", err: err}
		}
		if resp != nil && resp.NoData {
			return addValidatedMsg{symbol: symbol, noData: true, meta: resp}
		}
		return addValidatedMsg{symbol: symbol, quote: quote, meta: resp}
	}
}
