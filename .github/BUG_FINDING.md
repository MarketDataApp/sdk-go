# Bug Finding Workflow

This document defines a systematic process for discovering bugs in
`MarketDataApp/sdk-go` through exploration and testing, before users hit them.

> **IMPORTANT: Every bug found MUST be submitted as a GitHub issue.**
>
> Do not just record bugs in markdown files, notes, or comments. Each bug must become a
> real GitHub issue:
>
> - **CLI**: `gh issue create --label "bug" --title "[Bug]: ..." --body "..."`
> - **Web**: [Create Bug Report](https://github.com/MarketDataApp/sdk-go/issues/new?template=bug.yml)
>
> A bug hunt is not complete until every discovered bug exists as a GitHub issue.

## Overview

**Purpose**: proactive bug discovery, as opposed to reactive bug processing.

- **BUG_FINDING.md** (this document): find bugs before users encounter them.
- **[ISSUE_WORKFLOW.md](./ISSUE_WORKFLOW.md)**: process bug reports that users submit.

**Workflow**: Find Bug → **Create GitHub Issue (REQUIRED)** → [ISSUE_WORKFLOW.md] → Fix

**When to use this document**:

- QA passes before releases
- Pre-release validation (see [RELEASE_PROCESS.md](./RELEASE_PROCESS.md))
- Exploratory testing sessions
- After a significant refactor
- When onboarding, to understand the edge cases

---

## Prerequisites

### Environment setup

```bash
go build ./...
go version                # 1.22 or later; CI runs 1.22 and stable

export MARKETDATA_TOKEN="your_token_here"
```

### Baseline verification

Confirm the suite passes before hunting. Bug finding assumes a working baseline.

```bash
go test -race -p 1 ./...
```

> Run the race suite **sequentially** (`-p 1`). Running it in parallel across several
> sessions has previously exhausted memory on developer machines.

If tests fail, fix that first.

### Architecture

Familiarize yourself with the main components:

- `marketdata.Client` — entry point, from `marketdata.NewClient(opts...)`. Exposes
  `Stocks`, `Options`, `Funds`, `Markets`, `Utilities`. There is no global client and no
  `init()` side effect; the caller owns it and calls `Close()`.
- `internal/http` — transport, retries, the cached `/status/` retry gate, rate-limit
  reservations, response-size limits.
- The configuration cascade — `.env` < environment < client options < method parameters.
  **The method wins.**
- `*Response` — `Body`, `NoData`, `StatusCode`, `IsJSON`/`IsCSV`/`IsHTML`, `SaveToFile`,
  `String`.
- The error taxonomy in `marketdata/errors.go` — `AuthenticationError`,
  `BadRequestError`, `NotFoundError`, `RateLimitError`, `ServerError`, `NetworkError`,
  `ParseError`, `ValidationError`, `ResponseTooLargeError`, and more, each with a
  matching `Err*` sentinel. All of them satisfy the `marketdata.Error` interface, which
  is what exposes `Retryable()` and `SupportInfo()`.
- The sealed date-window union — `OnDate`, `Between`, `Since`, `Until`, `LastN`,
  `LastNUntil`, re-exported by each service package (`stocks.LastN(30)`,
  `funds.Between(a, b)`). It lives in `internal/params`, so callers reach it only
  through those aliases. Only one window can be supplied, and that is enforced at
  compile time (ADR-017).

### Two toolchains, always

`go.mod` declares `go 1.22`, and CI runs the suite on **1.22 and stable**. A bug that
appears on only one is still a bug, and is often a more interesting one. When a finding
looks toolchain-dependent, record which you observed it on.

```bash
GOTOOLCHAIN=go1.22.12 go test ./...
go test ./...
```

### Known differences, already tracked

Before filing anything involving live data, read `integration/discrepancy_test.go`.
Several API-vs-documentation differences are recorded there with links to the API's own
issue tracker. Re-reporting one of those wastes a cycle; finding a *new* one is valuable.

---

## Area 1: Error Handling and the Error Taxonomy

### What can go wrong

- The wrong error type for a given HTTP status
- `SupportInfo()` missing the request id or URL, making triage impossible
- An API token leaking into a message, a log line, or `SupportInfo()`
- A wrapped cause lost, so `errors.Is`/`errors.As` stop matching
- `ParseError` returned for a payload the SDK should handle
- An error that is genuinely retryable reporting `Retryable() == false`, or the reverse

### Test scenarios

#### 1.1 Bad token

```go
client, err := marketdata.NewClient(marketdata.WithToken("obviously-invalid-token"))
// Verify: an *AuthenticationError, and the token does NOT appear in err.Error().
// Bug indicator: a bare error string, or the token echoed back in any output.
```

#### 1.2 Unknown symbol

```go
q, resp, err := client.Stocks.Quote(ctx, "ZZZZQQ")
// Verify: a *NotFoundError. The SDK deliberately reports an unknown symbol as
// NotFoundError rather than as a successful empty answer.
// Bug indicator: err == nil with resp.NoData true, or a nil-pointer panic on q.
```

#### 1.3 SupportInfo completeness

```go
_, _, err := client.Stocks.Candles(ctx, "AAPL", stocks.WithCandleWindow(stocks.LastN(-5)))

// marketdata.Error is the interface every SDK error implements; it is what
// carries SupportInfo(). Do NOT reach for *marketdata.APIError here — that is
// one specific error (HTTP said OK, the body said otherwise), not a base type.
var mdErr marketdata.Error
if errors.As(err, &mdErr) {
	fmt.Println(mdErr.SupportInfo())
	// Verify: request_id, request_url, status_code, timestamp, message and
	// exception_type all present, in that order, column-aligned.
	// Bug indicator: blank values, missing lines, or a token in request_url.
}
```

#### 1.4 errors.Is / errors.As discipline

```go
// Every typed error should match both its concrete type and its sentinel.
var authErr *marketdata.AuthenticationError
errors.As(err, &authErr)              // want true
errors.Is(err, marketdata.ErrAuthentication) // want true
// Bug indicator: one works and the other does not, or Unwrap() drops the cause.
```

### Red flags

- A raw `*url.Error` or `*http.Response` reaching the caller
- An API token visible anywhere in output, including DEBUG logs
- `SupportInfo()` lines missing or unaligned
- A wrapped cause dropped where an underlying failure clearly existed

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Bad token | `*AuthenticationError`, token redacted | Generic error, or token leaked |
| Unknown symbol | `*NotFoundError`, consistently | Nil-pointer panic, or a silent empty success |
| SupportInfo | All six fields present and aligned | Blank or missing fields |
| Is/As | Both match, cause preserved | Either fails, or `Unwrap()` returns nil |

---

## Area 2: Empty and Sparse Responses

This area has the worst track record in the repo's history. Treat it as the highest-yield
place to look.

### What can go wrong

- A nil slice or nil pointer returned where an empty value is documented
- `NoData` disagreeing with what was actually returned
- A caller trusting `resp.NoData` (as the docs instruct) and then dereferencing nil
- The API omitting an array the decoder assumes is present
- A no-data chunk poisoning a merged multi-part response

### Test scenarios

#### 2.1 Empty result window

```go
// A valid question with an empty answer — a market holiday, or a window before listing.
candles, resp, err := client.Stocks.Candles(ctx, "AAPL",
	stocks.WithCandleWindow(stocks.OnDate(someMarketHoliday)))
// Verify: err == nil, resp.NoData == true, candles is empty and NOT nil.
// Bug indicator: a nil slice a range would silently skip, or a nil pointer.
```

#### 2.2 Empty list on a pointer-returning method

```go
exp, resp, err := client.Options.Expirations(ctx, "AAPL")
// Verify: on a 200 with an empty list, exp is non-nil with an empty Dates slice.
// The documented contract is "nil only on 404". A caller following the docs and
// checking resp.NoData instead of exp != nil must not panic.
```

#### 2.3 Missing optional fields

```go
// Sparse rows: a field the API omits entirely, versus one it sends as null.
// Verify: a null numeric stays distinguishable from a real 0 where the type
// allows it (earnings EPS is the precedent), and an absent array decodes to
// empty rather than panicking.
```

#### 2.4 Multi-part merge with a no-data part

```go
// A window wide enough to be split into chunks, where one chunk has no data.
candles, _, err := client.Stocks.Candles(ctx, "AAPL",
	stocks.WithCandleWindow(stocks.Between(longAgo, today)))
// Verify: the no-data chunk is dropped, not merged in as empty rows, and the
// surviving chunks are contiguous with no duplicated boundary day.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Empty window | `NoData` true, empty non-nil result | Nil slice or pointer |
| Empty list | Non-nil with empty slice | Nil, panicking a docs-following caller |
| Sparse fields | Null and zero stay distinct | Both collapse to zero |
| Merged parts | Contiguous, no duplicates | Duplicated boundary rows |

---

## Area 3: Concurrency, Rate Limits, and Composite Requests

### What can go wrong

- Rate-limit metadata from one request observed on another
- A retry consuming no reservation, so the limiter under-counts
- A reservation leaked when a request panics or is cancelled
- A cancelled fan-out reporting a deadline error as the root cause
- Chunk boundaries overlapping, duplicating rows

### Test scenarios

#### 3.1 Rate-limit metadata is request-scoped

```go
// Fire many requests concurrently; read resp.RateLimit on each.
// Verify: each response carries ITS OWN remaining count, not a shared latest value.
// Bug indicator: all responses reporting the same number.
```

#### 3.2 Reservation accounting across retries

```go
// Force a retryable 5xx so the SDK retries.
// Verify: every attempt reserves a credit, not just the first call, and the
// response's remaining count is recorded before the reservation is released.
// Bug indicator: the tracker drifting from the API's own header over a run.
```

#### 3.3 Cancellation during a fan-out

```go
ctx, cancel := context.WithCancel(context.Background())
// Cancel mid-flight during a multi-symbol call.
// Verify: the FIRST real error is what surfaces, not a downstream deadline
// error from a sibling request; cancellations log at DEBUG, not ERROR.
```

#### 3.4 Close() under concurrency

```go
// Call Close() while requests are in flight, and with a caller-injected
// WithHTTPClient transport.
// Verify: no panic, and the caller's own transport is never reached into —
// the SDK must not close idle connections it does not own.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Rate-limit scope | Per-response values | One shared value |
| Retry accounting | One reservation per attempt | One per call |
| Cancellation | First error surfaces, DEBUG log | Deadline error masks the cause, ERROR spam |
| Close | Clean, caller's transport untouched | Panic, or caller's pool disturbed |

---

## Area 4: Date, Time, and Number Handling

### What can go wrong

- A timestamp rendered in the host timezone rather than US/Eastern
- A null timestamp becoming the Unix epoch instead of the zero time
- A wire fraction (`0.052`) displayed as a percentage without conversion, or twice
- A countback window returning n+1 rows because the anchor is inclusive
- A year or DST boundary off by one day

### Test scenarios

#### 4.1 Host timezone independence

```bash
TZ=Australia/Sydney go test ./...
TZ=UTC              go test ./...
TZ=America/New_York go test ./...
# Verify: identical results. The SDK normalizes to US/Eastern and embeds tzdata,
# so it must not read the host zone.
```

#### 4.2 Date format round-trip

```go
// Compare the same call with the default format and with WithDateFormat.
// Verify: the same instants, differently rendered — not different instants.
```

#### 4.3 Null timestamp

```go
// A row whose timestamp the API sends as null.
// Verify: the zero time.Time, and t.IsZero() == true.
// Bug indicator: 1970-01-01, which is indistinguishable from a real epoch value.
```

#### 4.4 Countback row counts

```go
c, _, _ := client.Stocks.Candles(ctx, "AAPL", stocks.WithCandleWindow(stocks.LastN(5)))
// Verify: exactly 5 rows.
// Note: markets/status countback returning n+1 is a KNOWN API-side defect,
// already tracked. Anywhere else, n+1 is a new bug.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Host timezone | Identical under any `TZ` | Results shift with the host |
| Format round-trip | Same instants | Different instants |
| Null timestamp | Zero time, `IsZero()` | Unix epoch |
| Countback | Exactly n | n+1, or n-1 |

---

## Area 5: Configuration Cascade

### What can go wrong

- A `.env` value overriding a real environment variable
- A client option ignored when a method parameter is absent, or the reverse
- An explicitly-empty environment variable falling back to `.env`
- An environment variable documented but never read

### Test scenarios

#### 5.1 Method beats client

```go
client, _ := marketdata.NewClient(
	marketdata.WithToken(tok), marketdata.WithDateFormat("timestamp"))
// Pass a different date format at the method level.
// Verify: the method's value wins.
```

#### 5.2 Environment beats .env

```bash
# With a .env containing MARKETDATA_TOKEN=from_dotenv
MARKETDATA_TOKEN=from_env go run .
# Verify: from_env wins. A .env must never override a real environment variable.
```

#### 5.3 An explicitly empty variable is honored

```bash
MARKETDATA_TOKEN= go run .
# Verify: the empty value is respected (the CI pattern for forcing demo mode)
# and does NOT silently fall back to a .env file's value.
```

#### 5.4 WithoutDotEnv

```go
marketdata.NewClient(marketdata.WithToken(tok), marketdata.WithoutDotEnv())
// Verify: no .env file is read at all.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Method vs client | Method wins | Client wins, or method ignored |
| Env vs .env | Env wins | `.env` overrides a real variable |
| Empty env var | Honored as empty | Falls back to `.env` |
| `WithoutDotEnv` | No file read | `.env` still consulted |

---

## Area 6: Output Formats and File Export

### What can go wrong

- `IsJSON`/`IsCSV` disagreeing with the actual body
- A CSV facet forcing a parameter the caller did not ask for
- The columns projection dropping the field a row count keys off
- `SaveToFile` writing outside the intended path, or with unsafe permissions
- `String()` dumping an entire body where a summary is intended

### Test scenarios

#### 6.1 Format detection

```go
csvResp, err := client.Stocks.AsCSV().Candles(ctx, "AAPL")
// Verify: resp.IsCSV() true, IsJSON() false, and Body parses as CSV.
// Bug indicator: format flags set from the request rather than the response.
```

#### 6.2 Columns projection

```go
client, _ := marketdata.NewClient(marketdata.WithToken(tok),
	marketdata.WithColumns("close"))
// Verify: the SDK still requests whatever column it needs to count rows, and
// the decoded result is coherent rather than silently empty.
```

#### 6.3 SaveToFile

```go
_, resp, _ := client.Stocks.Candles(ctx, "AAPL")
err := resp.SaveToFile("/tmp/out.csv")
// Verify: the file lands exactly where asked, with sane permissions, and a
// path the caller did not intend is not reachable via the extension logic.
```

#### 6.4 The HTML facet stays unexported

```go
// ADR-018: HTML is built but deliberately NOT public, because the API does not
// support format=html yet.
// Bug indicator: an exported AsHTML() appearing on any service.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Format detection | Flags match the body | Flags reflect the request |
| Columns | Coherent result | Silently empty rows |
| SaveToFile | Intended path, sane perms | Escapes the path, or world-writable |
| HTML facet | Unexported | Publicly reachable |

---

## Area 7: Compile-Time Exclusivity

Go-specific, and the counterpart to the C# SDK's dependency-injection area. ADR-017 makes
mutually-exclusive parameters single sealed-union values, so illegal combinations do not
compile.

### What can go wrong

- A new option accepting a combination that should be impossible
- The negative-compile corpus asserting a *different* failure than the one intended
- A legal combination accidentally made to fail

### Test scenarios

#### 7.1 The corpus still holds

```bash
go test ./internal/negcompile/...
# Verify: every negative case fails to build, every positive case builds, and
# each negative case's expected message names the identifier it is about.
```

#### 7.2 A new option

```go
// After adding any option, ask: does it duplicate something an existing sealed
// union already covers? If a caller could supply both, it belongs in the union,
// not alongside it — and the corpus needs a new negative case.
```

### Pass/fail criteria

| Scenario | Pass | Fail |
|---|---|---|
| Corpus | All negatives fail, positives build | A negative compiles |
| New option | Folded into a union, corpus extended | Two options can conflict at runtime |

---

## Reporting What You Find

For every bug, open an issue immediately. Include:

1. The area and scenario number from this document
2. Minimal reproduction code, complete with the `import` block and `NewClient`
3. Expected versus actual behavior
4. The `SupportInfo()` block when an error was involved
5. SDK version, `go version`, and `go env GOOS GOARCH`
6. Whether it reproduces on Go 1.22, on stable, or on both

```bash
gh issue create --label "bug" \
  --title "[Bug]: Candles returns a nil slice for an empty window" \
  --body "$(cat <<'EOF'
**Area**: 2.1 Empty result window
**Reproduces on**: go1.22.12 and stable
...
EOF
)"
```

Then hand off to [ISSUE_WORKFLOW.md](./ISSUE_WORKFLOW.md).

---

## Coverage Note

This repo enforces **100% statement coverage** of `marketdata/...` and `internal/...`, so
every code path already has *a* test. That is exactly why this document targets behavior
rather than reachability: full coverage proves each line ran, not that it did the right
thing.

The bugs left to find here are wrong answers on covered lines, disagreements between the
two toolchain legs, assumptions about the host environment, and — most often in this
repo's history — the difference between "no data" and "nil". Note also that coverage is
scoped to the SDK packages: `examples/` is exercised by its own tests and the cross-app
method guard, not by the gate.
