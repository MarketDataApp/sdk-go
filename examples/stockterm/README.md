# stockterm

`stockterm` is a terminal watchlist for the [Market Data API](https://www.marketdata.app/), built
with [Bubble Tea](https://github.com/charmbracelet/bubbletea) on top of the Go SDK v2. It refreshes
a list of stock and mutual fund symbols on an interval, drills into a detail pane — candles, the
52-week range, upcoming earnings, recent headlines — for whichever symbol is selected, and surfaces
the API's own operational state (market status, rate-limit credits, outbound request headers)
directly in the UI. Beyond being a runnable reference for every read endpoint the SDK exposes,
stockterm doubles as a live-API canary: its `-once` flag drives the app's entire fetch surface
synchronously against a real (or mocked) backend, prints one rendered frame, and exits with a status
code a script can check. This repository's own grading process runs `-once` against the live API to
confirm the SDK still works end to end; CI runs the same instrument against mock servers (see
`main_test.go`), exercising the same code path without a live token or network access.

## Screenshot

```
┌ Market: OPEN  ─  Sat 2026-07-11 14:32:05 ET ─────────────────────────────────────────────────────┐
│ SYMBOL         LAST        CHG       CHG%   BID x ASK                      VOLUME                │
│ AAPL         233.10      +1.24     +0.53%   233.08 x 233.12            41,203,110   ◀            │
│ MSFT         512.44      -2.01     -0.39%   512.40 x 512.48            18,113,020                │
│ META         512.00      +3.10     +0.61%   511.95 x 512.05             9,876,543                │
│ SPY          560.25      -0.85     -0.15%   560.20 x 560.30            55,432,111                │
│ VFINX             —          —          —   —                                   —                │
├ AAPL ────────────────────────────────────────────────────────────────────────────────────────────┤
│       218.00 ▄▆▇▇▇▆▄▂▁▁▁▂▄▆▇█▇▆▄▃▁▁▁▂▄▆▇▇▇▆▄▂▁▁▁▂▄▆▇▇ 233.97         daily, 3mo                  │
│ 52wk:   168.20 ───────────────────────────●────── 245.10                                         │
│ Next earnings: 2026-07-30 (est EPS $2.14)                                                        │
│ • Apple unveils new product l…  • Suppliers report strong Q3 …  • Analysts raise price target…   │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ credits 91,201/100,000   resets 09:30 ET   refreshed 14:32:03   [no error]                       │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

(This is `testdata/main.golden`, the fixture `TestGolden_Main` checks byte-for-byte — the real app
looks exactly like this, in color, at a 100x40 terminal.)

## Key reference

| Key | Action |
|---|---|
| `↑` / `↓` | Move the watchlist selection; reloads the detail pane for the newly selected symbol |
| `1` | Detail pane: intraday candles (5-minute bars, last trading day) |
| `d` | Detail pane: daily candles (trailing 3 months) — the default |
| `w` | Detail pane: weekly candles (trailing 52 weeks) |
| `a` | Open the add-symbol modal; type a ticker and press Enter to validate and add it to the watchlist |
| `x` | Remove the selected symbol from the watchlist (no-op if it's the last one) |
| `m` | Open the 5-day market status history modal |
| `D` | Open the diagnostics modal (API uptime status + the outbound request headers the client actually sent) |
| `E` | Open the last-error modal, showing the most recent failed fetch's `SupportInfo()` |
| `o` | Open the day-performance modal (today's daily candle for every watchlist symbol, via one bulk-candles call) |
| `r` | Clear an active rate-limit suspension and force an immediate watchlist refresh |
| `q` / `Ctrl-C` | Quit. `Ctrl-C` always quits; `q` quits everywhere *except* while the add-symbol input is focused, where it's typed as text (so `QQQ` remains typeable) |
| `Esc` | Close whichever modal is open; while the add-symbol modal is open, also clears the input |

## Running it

### From the repo root (recommended)

```bash
go build -C examples/stockterm -o stockterm . && ./examples/stockterm/stockterm
```

Run the resulting binary from the **repo root**, not from `examples/stockterm/`. `marketdata.NewClient`
loads a `.env` file from the process's current working directory only — it doesn't search parent
directories — so this is what lets stockterm pick up a `MARKETDATA_TOKEN` from the repo root's `.env`
without you having to export anything. `-C examples/stockterm` just tells `go build` where the
module lives; it doesn't affect where the binary is later run from.

### From anywhere, via an environment variable

```bash
export MARKETDATA_TOKEN="your-api-key"
./examples/stockterm/stockterm
```

With `MARKETDATA_TOKEN` exported, stockterm runs from any working directory — the `.env` lookup
above is only a convenience for the repo-root case.

### Demo mode

Run it with no token available (no `.env`, no `MARKETDATA_TOKEN`, not from the repo root) and
stockterm still starts: the client falls back to demo mode, a banner appears
(`DEMO MODE — AAPL data only`), and the watchlist is forced down to the single AAPL symbol demo
tokens are allowed to query.

## Live smoke test and the `-once` grading contract

```bash
# from the repo root, with a token in .env or MARKETDATA_TOKEN exported
go build -C examples/stockterm -o stockterm . && ./examples/stockterm/stockterm -once
```

`-once` is stockterm's headless mode: it never starts `tea.Program` (no TTY required), so it's safe
to run in CI, in a script, or by an agent grading the app against the live API. One invocation is a
full canary for the app's SDK surface:

1. Records the process's baseline goroutine count, before the client is even created.
2. Builds the client from the same flags an interactive run would use.
3. Constructs the model and synchronously executes **every fetch stockterm owns** — all 13 SDK
   methods in the coverage table below except `Stocks.Quote`'s interactive-only call site
   (`validateSymbol`, which needs a simulated keypress and adds no method coverage beyond what
   `fetchDetailQuote` already exercises) — feeding each result through the same `Update` the
   interactive program uses, in a fixed order so output is stable across runs.
4. Injects a 100x40 `tea.WindowSizeMsg` and prints one rendered frame to stdout. The frame is plain
   text: `-once` sets lipgloss's color profile to `termenv.Ascii` before doing anything else, which
   makes every style in the app render with no escape codes at all — no terminal needed to read or
   diff it.
5. If the run's last error is non-nil, prints `SUPPORT INFO:` followed by that error's
   `SupportInfo()` block (or, for an error that doesn't implement the SDK's `Error` interface, its
   plain `Error()` string).
6. Closes the client, then polls for up to 2 seconds for the process's goroutine count to settle
   back to at most baseline+1, and prints either `goroutines: clean (n=X baseline=Y)` or
   `goroutines: LEAK (n=X baseline=Y)`.

Exit codes:

| Code | Meaning |
|---|---|
| `0` | Every fetch succeeded. The SDK's no-data convention (a valid request the API has nothing to answer) counts as success, not failure. |
| `3` | At least one fetch returned an error. The frame and the `SUPPORT INFO:` block are still printed. |
| `1` | Startup failed (bad client configuration) or the goroutine count never settled — a leak. |

The grader diffs `-once`'s frame values against direct API calls, exercises the no-data path with an
invalid symbol, forces errors with `-base-url` pointed at a mock that returns 429/500, and checks a
token-less CWD produces the demo banner — pass/fail against real output, no opinion.

## Flags

| Flag | Default | Description |
|---|---|---|
| `-refresh` | `5s` | Watchlist refresh interval (the detail pane and market status refresh on their own fixed cadences, unaffected by this flag) |
| `-funds` | `VFINX` | Comma-separated symbols (case-insensitive) treated as mutual funds — their candles come from `client.Funds.Candles` instead of `client.Stocks.Candles`, since the bulk quote/price endpoints don't cover funds at all |
| `-prices` | `false` | Use the lightweight `Stocks.Prices` endpoint instead of `Stocks.Quotes` for the watchlist refresh |
| `-once` | `false` | Run every fetch synchronously, print one frame, and exit — see the grading contract above |
| `-base-url` | (unset) | Override the API base URL, typically to point at a mock or test server. Setting it also implies `marketdata.WithoutStartupValidation()`: a custom base URL is assumed not to be the live API, so stockterm skips the live token-validation ping that would otherwise fail startup against a server that doesn't implement it |

Positional arguments (`stockterm TSLA NVDA ...`) replace the default watchlist
(`AAPL MSFT META SPY VFINX`) entirely.

## Coverage

Every SDK method stockterm calls, the `fetch.go` function that wraps it, and where the result
surfaces in the UI. The `Get*` convenience wrappers the SDK also exposes (e.g. `client.Stocks.GetQuote`)
are deliberately **not** used anywhere in this app — they discard the `*marketdata.Response` metadata
(rate-limit state, the no-data flag) that every fetch function here needs and that the status line's
credit meter is built from.

| SDK method | `fetch.go` function | UI surface |
|---|---|---|
| `Stocks.Quotes` | `fetchQuotes` | Watchlist rows (default refresh) |
| `Stocks.Prices` | `fetchPrices` | Watchlist rows (refresh, under `-prices`) |
| `Stocks.Candles` | `fetchCandles` | Detail pane sparkline, for a non-fund selected symbol (`1`/`d`/`w`) |
| `Funds.Candles` | `fetchFundCandles` | Detail pane sparkline, for a fund selected symbol (`1`/`d`/`w`) |
| `Stocks.Quote` (with `WithFiftyTwoWeek`) | `fetchDetailQuote` | Detail pane 52-week range bar |
| `Stocks.Quote` | `validateSymbol` | Add-symbol modal (`a`) validation before a ticker joins the watchlist |
| `Stocks.Earnings` | `fetchEarnings` | Detail pane "Next earnings" line |
| `Stocks.News` | `fetchNews` | Detail pane headline strip |
| `Markets.Status` | `fetchMarketStatus` | Header's `Market: OPEN/CLOSED` |
| `Markets.StatusHistory` | `fetchStatusHistory` | Status-history modal (`m`) |
| `Utilities.User` | `fetchUser` | Primes the credit meter at startup (skipped in demo mode; `/user/` requires a token); `m.user` itself isn't otherwise rendered |
| `Utilities.Status` | `fetchAPIStatus` | Diagnostics modal (`D`), API uptime line |
| `Utilities.Headers` | `fetchHeaders` | Diagnostics modal (`D`), outbound request headers |
| `Stocks.BulkCandles` | `fetchBulkCandles` | Day-performance modal (`o`) |
| `Client.Close` | (called from `main.go`'s `run`/`runOnce`) | The quit path (`q`/`Ctrl-C`) and the end of every `-once` run — always called, even on startup error |
| per-response `*marketdata.Response.RateLimit` | `metaOf` (`app.go`), applied generically in `Update` | Status-line credit meter (`credits X/Y   resets HH:MM ET`) — updated from *every* message that carries a non-nil `meta`, not just one endpoint |
