# optionterm

An interactive terminal explorer for option chains, built on the
[Market Data Go SDK v2](https://github.com/MarketDataApp/sdk-go). It is two
things at once:

- **Reference code.** Every SDK call the program makes lives in `fetch.go`,
  one function per operation, following the same shape every time: a
  timeout-bounded context, exactly one SDK method call, an `errMsg` on
  failure, a typed message carrying the result plus `*marketdata.Response`
  metadata on success. Copy any function out of `fetch.go` verbatim into
  your own program and it will work.
- **A live canary.** `-once` runs the program's entire fetch surface
  synchronously against the real API (or a mock, via `-base-url`), prints
  one frame, and reports whether the session leaked goroutines — the
  fastest way to prove the SDK still behaves the way this app expects it
  to, without a TTY or a human watching a refresh loop.

## Screenshot

```
┌ AAPL  233.10  ─  options chain ──────────────────────────────────────────────────────────────────┐
│ EXPIRATIONS │    STRIKE      BID    ASK    MID   LAST     VOL      OI    IV                      │
│ 2026-07-17  │  • 225 C      9.10   9.25   9.18   9.20   3,201  12,410  0.28                      │
│ 2026-07-24  │  • 230 C      5.40   5.55   5.48   5.50   8,113  22,081  0.27                      │
│▶2026-08-21  │▶A  233 C      3.05   3.15   3.10   3.12  11,402  31,220  0.26                      │
│  (41 DTE)   │  • 235 P      4.85   4.95   4.90   4.88   6,240  18,377  0.27                      │
│ 2026-09-18  │                                                                                    │
│ 2026-10-16  │ ── PINNED ─────────────────────────────────────────────────────────────────────────│
│             │  AAPL260821C00233000  3.10  Δ0.52   AAPL260918P00230000  2.15  Δ-0.31              │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ credits 91,180/100,000   [/] lookup  [g] greeks  [c/p/b] side  [+/-] rng                         │
│ [no error]                                                                                       │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

(Verbatim from `testdata/main_both_sides.golden` — the fixture the
golden-frame tests in `views_test.go` pin against, not a live capture.)

## Key reference

| Key | Action |
|---|---|
| `tab`, `←`/`→` | switch focus between the expirations sidebar and the chain table |
| `↑`/`↓` | move the selection in whichever pane has focus |
| `enter` | open the contract-detail modal for the selected row |
| `esc` | close the top modal (detail, then support info), or cancel an open input |
| `space` | pin/unpin the selected row |
| `/` | open the lookup input (`SYMBOL YYYY-MM-DD STRIKE call\|put`) |
| `s` | open the symbol input to switch underlyings |
| `c` / `p` / `b` | filter the chain to calls / puts / both |
| `g` | toggle market-data columns vs. the Greeks |
| `+` / `-` | widen / narrow the strike window (2%–50%, ±5% per press) |
| `E` | open the support-info modal for the last error |
| `r` | force an immediate refresh (also clears a rate-limit suspension) |
| `q`, `ctrl+c` | quit |

## How to run

Build and run from the **repository root**, not from inside
`examples/optionterm`:

```bash
go build -C examples/optionterm -o optionterm . && ./examples/optionterm/optionterm
```

This matters because `marketdata.NewClient()` loads a `.env` file from the
process's current working directory. Running the binary from repo root (as
above) lets it pick up a repo-root `.env` with `MARKETDATA_TOKEN` set;
running it from inside `examples/optionterm` will not see that file unless
one exists there too.

**Env-var alternative** (works regardless of where `.env` lives — note
`examples/optionterm` is its own Go module, so `go run` needs to be invoked
from inside it, not with a package path from repo root):

```bash
export MARKETDATA_TOKEN="your-api-key"
cd examples/optionterm && go run . -symbol AAPL -refresh 15s
```

**Demo mode**: with no token found anywhere, the client falls back to a
fixed demo dataset and the app forces the underlying symbol to `AAPL`
regardless of `-symbol`, showing a persistent `DEMO MODE — AAPL data only`
banner. This is the zero-setup path — just run the binary with no token
configured.

## Live smoke test

```bash
go build -C examples/optionterm -o optionterm . && ./examples/optionterm/optionterm -once
```

This is `-once` mode: it builds a client, drives the program's full
six-operation fetch surface synchronously (no `tea.Program`, no TTY),
prints one frame to stdout, and exits. It is both the fresh-context
grader's instrument and the fastest manual check that this app still
matches the live API.

`-once` fetch order:

1. `fetchUnderlying` — the underlying quote.
2. `fetchExpirations` — the expiration list.
3. `fetchChain` — the chain for the nearest future expiration, first load
   (no strike filter yet — the filter only applies once a chain response
   has reported an underlying price).
4. If the chain returned at least one row: `fetchContract`,
   `fetchPinned`, and `lookupContract` for the at-the-money row, in that
   order — exercising the remaining three options endpoints against a
   contract the chain just proved exists.

A no-data response at any step (an invalid symbol, or a chain with nothing
in range) is not an error: it just leaves nothing for the later steps to
act on, so they are skipped rather than failing. `-once` output:

- The rendered frame (a 100×40 `tea.WindowSizeMsg`, ANSI-free — colors are
  forced to `termenv.Ascii` since the output is meant for a pipe or a
  diff, not a terminal).
- `SUPPORT INFO:` followed by the failing error's `SupportInfo()` block,
  printed only if at least one fetch failed outright.
- A final `goroutines: clean (n=X baseline=Y)` or `goroutines: LEAK (...)`
  line, from polling `runtime.NumGoroutine()` for up to 2 seconds after
  `client.Close()`.

Exit codes: `0` every fetch succeeded or was a documented no-data response;
`3` at least one fetch failed (the frame and `SUPPORT INFO:` are still
printed); `1` the client itself could not be constructed, or the session's
goroutines never settled.

## Flags

| Flag | Default | Description |
|---|---|---|
| `-symbol` | `AAPL` | underlying stock symbol for the option chain |
| `-refresh` | `15s` | chain refresh interval (interactive mode) |
| `-once` | `false` | fetch once, print a frame, and exit — no TTY required |
| `-base-url` | *(empty)* | override the API base URL, for testing against a mock server. Setting it automatically applies `marketdata.WithoutStartupValidation()`, so the mock does not need to simulate token validation |

## SDK coverage

Every SDK call this program makes, where it is made, and where its result
shows up on screen. `Get*` convenience wrappers (`GetChain`, `GetQuote`,
`GetQuotes`, `GetLookup`, `GetExpirations`, `stocks.GetQuote`, ...) are
deliberately never used: they discard the `*marketdata.Response` each call
returns, and this app's footer needs that response's rate-limit metadata
on every request, not just an aggregate snapshot.

| SDK method | `fetch.go` function | UI surface |
|---|---|---|
| `client.Stocks.Quote` | `fetchUnderlying` | top border: underlying symbol + last price |
| `client.Options.Expirations` | `fetchExpirations` | expirations sidebar (▶ selected, `(NN DTE)`) |
| `client.Options.Chain` | `fetchChain` | chain table (STRIKE/BID/ASK/MID/LAST/VOL/OI/IV, or the Greeks with `g`) |
| `client.Options.Quote` | `fetchContract` | contract-detail modal (`enter` on a row) |
| `client.Options.Quotes` | `fetchPinned` | pinboard strip (`space` to pin/unpin) — a concurrent fan-out, one request per pinned symbol, drawing from the client's shared 50-slot concurrency pool rather than a single batched call |
| `client.Options.Lookup` | `lookupContract` | lookup input (`/`), resolves an OCC symbol which is then fed into `fetchContract` to open the detail modal |
| `client.Close` | `main`, `once.go`'s `runOnce` | not user-visible; releases idle HTTP connections on exit (interactive quit or `-once`'s return) |
| every response's `RateLimit` metadata | `applyMeta` (`app.go`), folded from every success message's `meta` | footer credits meter (`credits remaining/limit`) |
