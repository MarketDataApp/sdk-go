# Examples

Runnable reference programs for the Market Data Go SDK v2. All of them read
`MARKETDATA_TOKEN` from the environment (or a `.env` file in the working
directory); without a token they run in demo mode, which the API restricts to
a small set of symbols (e.g. `AAPL`) with limited access.

Run any example from the repository root:

```bash
go run ./examples/<name> [flags]      # -h shows each example's flags
```

## Command-line examples

| Example | What it shows | SDK surface exercised |
|---|---|---|
| [basic](basic/) | Minimal quick start: quotes, market status, candles, 52-week data, rate-limit metadata | `Stocks.Quote/Quotes/Candles`, `Markets.Status`, `GetQuote` convenience wrappers |
| [covered-call-screener](covered-call-screener/) | Scans option chains for covered-call candidates (DTE, volume, open interest, spread filters) and ranks by annualized premium | `Options.Chain` with strike/expiry/side filters, concurrent fan-out |
| [earnings-analyzer](earnings-analyzer/) | Fetches recent earnings for several symbols and correlates surprises with price reaction | `Stocks.Earnings`, `Stocks.Candles`, optional `Stocks.News` |
| [historical-exporter](historical-exporter/) | Exports OHLCV candles to CSV files, one per symbol (`-outdir`), for stocks or funds | `Stocks.Candles`, `Funds.Candles`, date windows, automatic intraday range splitting |
| [multi-asset-dashboard](multi-asset-dashboard/) | One-shot dashboard mixing stocks, options, funds, market status, API status, and account info, fetched concurrently | Every service; concurrent use of one shared client |
| [response-formats](response-formats/) | The `*Response` most examples discard: the CSV facet, the `IsJSON`/`IsCSV`/`IsHTML` predicates, `Body()`, `SaveToFile()`, and how to total session credits correctly | `Stocks.Candles`, `Stocks.AsCSV().Candles`, `response.Response`/`CSVResponse` |
| [portfolio-monitor](portfolio-monitor/) | Terminal ticker: full per-symbol quotes (with 52-week data) at start, lightweight price polling afterwards | `Stocks.Quote` (52-week is single-quote only), `Stocks.Prices`, `Markets.Status` |
| [watchlist-alerter](watchlist-alerter/) | Polls a watchlist and fires alerts on price moves, wide spreads, and proximity to 52-week extremes | `Stocks.Quote` per symbol, typed error handling (`ErrRateLimited`) |

Credit note on the footers: every example prints `Credits remaining` plus what
the *last* request cost. `client.RateLimits()` is a snapshot of the most
recently completed response — its `Consumed` field is that one request's cost,
never a running total, and under concurrency it reports whichever request
finished last. To total a session, sum `resp.RateLimit.Consumed` across your own
calls, as [response-formats](response-formats/) does.

Credit note: bulk endpoints (`Quotes`, `Prices`) cost 1 credit per request
regardless of symbol count. 52-week data only exists on the single-quote
endpoint, so `portfolio-monitor` and `watchlist-alerter` spend 1 credit per
symbol per refresh — deliberate, and called out in their source.

## Terminal (TUI) applications

Full-screen Bubble Tea apps, each an independent Go module with its own
README, tests, and golden-file suite. Between them they exercise every
context-first SDK service method except three that are explicitly exempted
with a reason — enforced by [coverage_test.go](coverage_test.go), which
derives the method set from the SDK source rather than a fixed list.

| App | What it shows |
|---|---|
| [stockterm](stockterm/) | Watchlist browser: bulk quotes, per-symbol detail (52-week range), candles with sparklines, funds, market/API status |
| [optionterm](optionterm/) | Options chain explorer: expirations, chain filtering, per-contract quotes, lookup |
| [tuitest](tuitest/) | Shared headless test harness for driving the TUI apps in CI (not an SDK example) |

Because they are separate modules, run them from their own directory:

```bash
cd examples/stockterm && go run .
```
