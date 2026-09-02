# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [2.0.0] - 2026-09-02

Initial v2 release. A complete rewrite of the SDK following idiomatic Go
patterns; see docs/MIGRATION.md for migrating from v1.

### Added
- Stocks: Quote, Quotes, Candles, BulkCandles, Earnings, News, Prices
- Options: Chain, Expirations, Lookup, Quote, Quotes
- Funds: Candles
- Markets: Status, StatusHistory
- Utilities: Status, Headers, User
- Two call styles for every endpoint: context-first methods returning `(T, *Response, error)`, plus `Get*` convenience wrappers returning `(T, error)`
- Compile-time parameter exclusivity: mutually-exclusive parameters are single sealed-union values, so illegal combinations (e.g. a date range plus a countback) do not compile; enforced by the `internal/negcompile` suite (see ADR-017)
- Functional options with sealed-union values for date windows (`Between`/`Since`/`Until`/`OnDate`/`LastN`/`LastNUntil`) and option strike/expiry filters
- Universal parameters as client options: `dateformat`, `columns`, add-headers, human-readable, `mode`, `limit`, `offset`, `maxage`
- Typed error hierarchy with `errors.Is`/`errors.As` support and `SupportInfo()` for support tickets
- Per-response rate-limit metadata on the returned `*Response`, plus a client-level `RateLimits()` snapshot
- Automatic splitting of large intraday candle ranges into year-sized chunks, fetched concurrently and merged in chronological order
- US/Eastern timezone normalization for all timestamps
- `.env` file loading for project-level configuration
- 404 responses return NoData instead of errors
- `Close()` method for resource cleanup
- Demo mode for unauthenticated access
- Startup token validation with opt-out (`WithoutStartupValidation`)
- Uncapped exponential retry backoff (`1s * 2^retry` per SDK requirements §9.3), configurable with `WithMaxRetries`
- Comprehensive godoc for every exported identifier, with API documentation links and a package overview in `doc.go`
- Compile-checked godoc examples for every service method (`example_test.go` in each package)
- Executable examples with verified output in the integration suite (run against the live API with `-tags=integration`)
- Online SDK documentation at https://www.marketdata.app/docs/sdk/go

### Added (2026-08-04)
- Requirements-§7 log points: `client initialized` at INFO, terminal request failures at ERROR, and a `duration` field on the DEBUG response record
- `WithAPIVersion()` client option (completes the cascade over `MARKETDATA_API_VERSION`) and a public `marketdata.Version()` (ADR-015)
- Batch fan-outs (`Options.Quotes`, the intraday candle splitter) now cancel sibling requests on the first hard error (ADR-014), so a failing batch stops spending API credits; the root error is surfaced, never a cancellation echo
- README "String Conversion" section (ADR-016)
- Wire-contract tests: every endpoint now has a hand-written JSON fixture in its package's `testdata/` directory (as ADR-010 documents) asserted field by field against the public structs, so a wire-struct tag typo or swapped field fails offline. Also: a deterministic `RateLimits()` accounting test, a behavioral release-reservation-on-failure test, a 120-requests-over-50-slots concurrency-pool test, and channel-synchronized status-cache tests (replacing ~700ms of fixed sleeps)
- `WithoutDotEnv()` option (promised by ADR-012): disables loading a `.env` file from the working directory, for controlled environments where implicit file-based configuration is unwanted

### Changed (2026-09-01, no-data contract) — BREAKING
- **An unknown symbol is now an error, not an empty answer.** The API answers both
  "your question was invalid" and "your valid question has an empty answer" with
  HTTP 404, separated only by an `errmsg` field. The SDK intercepted every 404 as
  no-data before the error mapping ran, so a typo'd or delisted ticker came back as
  a nil result with a nil error — a silently wrong answer. A 404 whose body names an
  error now maps to `marketdata.NotFoundError` (sentinel `marketdata.ErrNotFound`),
  on the typed methods and the CSV facets alike. A markerless 404 is unchanged: it
  is still no data, not a failure.
  Note the API does not mark every endpoint — `options/expirations` and
  `stocks/candles` answer an unknown symbol with an unmarked 404, indistinguishable
  on the wire from an empty result, so those still report no data.
- **The collection-shaped results return an empty value instead of nil on no-data.**
  `Options.Chain`, `Options.Expirations` and `Utilities.Headers` now return a non-nil
  empty result, matching what an empty HTTP 200 already produced, so ranging the
  result is safe regardless of which shape the API happens to send. Ranging the old
  nil panicked, which is how it reached CI.
- The five scalar-shaped results — `Stocks.Quote`, `Options.Quote`, `Markets.Status`,
  `Utilities.Status` and `Utilities.User` — deliberately still return nil on no-data,
  and callers should keep checking. A zero-valued one of these would read as real
  data (a price of zero, a closed market, an account with no credits), which is worse
  than an explicit nil. See the "Missing Data and Unknown Symbols" section of the
  package documentation.

### Changed (2026-08-20, post-audit remediation) — BREAKING
- `WithHumanReadable` no longer applies to the typed JSON methods; it reaches the CSV
  and HTML facets only. It renames every field in the response — "askSize" becomes
  "Ask Size", the "s" envelope field disappears — so a typed method receiving it failed
  outright. It was documented as having "no effect on the typed JSON methods", which was
  false in the worst direction: not inert, fatal.
- `WithColumns` now also requests the "s" envelope column on typed JSON requests. The API
  drops that field whenever a column filter is set and every typed method checks it, so
  the option broke every typed call. The formatted facets are unchanged and still receive
  exactly the columns asked for, since there the column list is the output.

### Added (2026-08-20, post-audit remediation)
- `marketdata.ResponseTooLargeError` and `marketdata.InsecureTokenError`, plus the
  `ErrResponseTooLarge` and `ErrInsecureToken` sentinels. Both types were already returned
  to callers but lived only under `internal/`, so `errors.As` on either was impossible
  from outside the module — a hole SemVer would have frozen at the tag.
- `Stocks.GetBulkCandles` and `Markets.GetStatusHistory`, the two missing halves of the
  "every service method has two forms" promise in the package overview.

### Fixed (2026-08-20, post-audit remediation)
- **A caller's `context.WithTimeout` produced one ERROR log line per fan-out sibling.**
  One expiry is reported by every sibling, so a 50-symbol batch logged 50 lines.
  Deadlines are now demoted only inside a fan-out; a standalone deadline still logs at
  ERROR, and a 401 or 500 on one sibling still logs — a distinct server answer is not an
  echo.
- **The four rate-limit headers were parsed four times by three different rules**, and
  disagreed on the same response: a reset of 0 read as the zero time on one public
  surface and as `1969-12-31` on two others — a rendering this release claims to have
  fixed, and had fixed in one of the four copies. One parser now serves all of them, with
  `ResetAt` normalized to US/Eastern as ADR-005 requires.
- **A response body dropped mid-read escaped the error taxonomy and was never retried.**
  It returned an untyped error, so `errors.As` missed it and it carried no support
  context, while `ShouldRetryError` said no — leaving the most retryable failure class
  unretried. It is now a `NetworkError` and is retried.
- **Every error reported a `request_url` that had never been sent**, rebuilt from the
  caller's params rather than the merged request, so a client's universal defaults —
  `columns`, `mode`, `limit` — were missing from the exact string meant to be pasted into
  a support ticket.
- **Retries spent up to four credits against one reservation.** The pre-flight reserve sat
  outside the retry loop while every attempt inside it sends a real, billed request.
- `multi-asset-dashboard`'s "Top Calls by Volume" table was in strike order: the closure
  named `sortByVolume` only truncated. `stockterm` and `optionterm` printed the same
  credits label with opposite meanings. `historical-exporter` wrote its CSVs into the
  working directory by default.
- Documentation: `docs/MIGRATION.md`'s import path was lowercase against a case-sensitive
  module path, so the guide's first instruction resolved to **v1**. Six options cited
  across CLAUDE.md, COMPARISON.md and MIGRATION.md do not exist. `WithMaxBidAskSpreadPct`
  documented percentage of the midpoint where the API applies percentage of the
  underlying; `WithLimit`/`WithOffset` read as universal when the candles endpoints ignore
  them; `markets.LastN` did not mention that this endpoint returns n+1 rows.

### Changed (2026-08-20, post-audit remediation)
- The parameter audit now asserts in both directions: every query key observed on the wire
  must be catalogued, not only that every catalogued parameter reaches the wire. Its view
  of "what the SDK sends" was a hand-written document, so a stray parameter was invisible
  — and `funds/candles`, exempt from the OpenAPI comparison, had no second witness at all.
- Six configuration constants that encode a policy decision are now pinned: pool size,
  the redirect and response-size caps, idle connections per host, the Retry-After ceiling
  and the `SaveToFile` file mode. All were inside the 100% coverage figure and compared
  nowhere; a mutation battery changed each without turning the suite red.
- The examples' SDK-coverage guard derives its method set from the source instead of a
  hand-written list of 18, which had frozen while the SDK grew to 21. The three uncovered
  methods are explicit exemptions, and a stale exemption is itself a failure.
- The offline audit fails when its checked-in OpenAPI snapshot is more than 90 days old;
  the freshness check previously required only that the date field be non-empty.
- The negative-compile corpus requires its `want` patterns to name the identifier under
  test rather than a bare diagnostic kind, which any compile error satisfied.

### Known API defects (2026-08-20, second round)
- `options/chain` applies a delta filter only on the call side: with `side=put`, or with
  no side, the parameter is ignored and the unfiltered chain returns at HTTP 200. This
  contradicts the absolute-value semantics `ByDelta` documents.
- The candles endpoints ignore the universal `limit` and `offset` parameters, which other
  endpoints honor. `offset` is the dangerous half — paging over candles returns the
  identical full set on every page.
- `markets/status-history` returns `countback+1` rows where every other endpoint's
  countback is exact. The SDK deliberately does not compensate: the API honors the
  parameter with an off-by-one rather than ignoring it, so adjusting would break silently
  once it is corrected.

### Changed (2026-08-20, post-release audit) — BREAKING
- `stocks.WithCandleExchange` and `stocks.WithCandleCountry` are **removed**. The API
  accepts both and ignores both — verified live, including `exchange=BASURATOTAL`
  returning the normal US candles — so they advertised a symbol-resolution control that
  was a no-op. Removed rather than kept as documented-inert options, in the pre-tag
  window. Callers relying on them were already getting the default exchange's data.

### Fixed (2026-08-20, post-release audit)
- **`Stocks.BulkCandles` silently discarded a valid response for a single symbol.** For
  one symbol the API omits the `symbol` key entirely, and the decoder keyed its row count
  off that array — so a request that returned (and billed for) a candle came back as an
  empty slice with a nil error and `NoData` false. Any loop that batches symbols and
  lands on a chunk of one hit it. The row count now comes from the timestamp array, and
  the symbol is restored from the request. The wire-contract fixture that concealed this
  — it carried the multi-symbol shape while the test asked for one symbol — was replaced
  with the two shapes production actually sends.
- **`Utilities.Status` reported the API online when the API reported an error.** It was
  the only decoding method without the `200`-but-not-`ok` guard, so a body of
  `{"s":"error"}` produced an `APIStatus` with `IsOnline() == true` and a nil error, on
  the one endpoint whose purpose is monitoring. An empty service list no longer counts as
  healthy either. Retry suppression was never affected: the offline gate reads only the
  status code.
- **`UserInfo.OptionsDataPermissions` was always empty.** The SDK read a body field the
  API always sends blank while ignoring the response header carrying the real value, so
  an account with full real-time entitlements reported none. It now falls back to the
  header, as the credit fields already did.
- **A caller's `context.WithTimeout` produced one ERROR log line per fan-out sibling.**
  The 2026-08-19 fix demoted cancellations but kept deadlines at ERROR; in a fan-out one
  expiry is reported by every sibling, so a 50-symbol batch logged 50 lines. Deadlines
  are now demoted only inside a fan-out. A standalone deadline still logs at ERROR, and a
  401 or 500 on one sibling still logs — a distinct server answer is not an echo.
- `WithStrikeLimit` is documented correctly: the API applies it per side of the money, so
  `n` admits up to `2n` distinct strikes. The godoc, the parameter catalog and an
  integration assertion all said `n`.
- The live integration suite failed on five catalogued parameters with no acceptance
  probe, and reported success anyway when no token was present — so in CI, with the
  secret unset, the whole job had been a silent no-op. It now fails loudly when `CI` is
  set and no token is found, and the missing probes were added. Separately, the suite's
  own token helper used a bare `os.Getenv`, so every probe that built its own client ran
  in demo mode; it now resolves through the SDK's configuration cascade.

### Known API defects (2026-08-20)
- `options/chain` reported `strike: 0` for every deep-in-the-money row of an unfiltered
  request (83 of 198 contracts on AAPL). **Resolved API-side on 2026-08-20**, roughly a
  day after it was observed; the strict assertion in `integration/discrepancy_test.go`
  now passes and is kept as a regression guard. The original report also claimed the
  defect propagated into the endpoint's strike filters — it did not: that evidence came
  from a run with an experimental OCC-derived strike active, and the filters work
  correctly when probed with the wire syntax the SDK actually sends (`strike=<=190`).

### Changed (2026-08-19, PR #33 review) — BREAKING
- `Options.AsCSV().Quotes` now takes `([]string, ...QuoteOption)` instead of variadic
  `...string`, matching the shape `Options.Quotes` took on 2026-08-12. The batch facet
  accepted no options at all, so a historical window was exportable as CSV for a single
  contract but not for a watchlist; gaining that after the tag would have been a second
  breaking signature change, post-release. Migration: wrap the symbols in a slice —
  `AsCSV().Quotes(ctx, "AAPL...C...", "AAPL...P...")` becomes
  `AsCSV().Quotes(ctx, []string{"AAPL...C...", "AAPL...P..."})`.
- `options.OptionQuoteOption` is now `options.QuoteOption`. The old name doubled the
  word — `Option` the instrument, `Option` the functional-option suffix — and was the
  only option type in the SDK not following the `<Method>Option` convention that
  `ChainOption`, `ExpirationOption` and `stocks.QuoteOption` already use. `OptionQuote`,
  `OptionQuoteWindow` and `WithOptionQuoteWindow` are unchanged.

### Fixed (2026-08-19, PR #33 review)
- **Intraday candle chunking dropped `exchange` and `country`.** A `Stocks.Candles`
  request for a non-US listing over more than a year at an intraday resolution returned
  the **US listing's prices**, with HTTP 200 and a nil error — while the same request at
  daily resolution, or over a shorter range, returned the correct listing. The chunk
  options were copied field by field and these two were left out; the chunk now copies
  the whole option set and narrows only the window, so a future option cannot be dropped
  the same way. (Not reported by the review; found while fixing the item below.)
- **Three parameters were silently dropped by the CSV and HTML facets** — `candle` on
  stock quotes, `exchange`/`country` on candles, and `report` on earnings. Each facet
  accepted the option and never sent it, so e.g.
  `AsCSV().Candles(ctx, "RY", WithCandleCountry("CA"))` returned the US listing with no
  error. Every endpoint had two request serializers — the JSON method and the facet
  builder — bound only by a comment promising they matched, and they had drifted inside
  this release. All 12 JSON methods now build their request through the same builder the
  facets use, so a second serializer no longer exists.
- **No-data chunks corrupted merged CSV output.** The API answers an empty window with
  HTTP 200 and a placeholder body carrying no marker, so those lines entered the merged
  text; and because the header dedup takes its reference from the first chunk, an empty
  *oldest* chunk (the likeliest, since it predates the listing) left a duplicate header
  row mid-file. Empty chunks are now detected structurally and dropped before the merge.
- **The CSV facet of `Options.Expirations` forced `dateformat=unix`**, so every CSV
  export returned `1787112000` instead of `2026-08-19`, and no `WithDateFormat` option or
  `MARKETDATA_DATE_FORMAT` value could change it. That parameter serves the JSON decoder;
  a facet returning the API's text verbatim must not set it.
- **Context cancellations logged at ERROR.** The cancel-on-first-error fan-outs cancel
  every sibling on the first failure, so one 401 in a 50-symbol batch put 50 ERROR lines
  and 88 cancellation echoes on stderr with zero configuration. Cancellations now log at
  DEBUG; `context.DeadlineExceeded` stays at ERROR, and the error is still returned to
  the caller unchanged.
- `GetFormatted` no longer mutates the caller's params map, which could leak `format=csv`
  into a later JSON request sharing that map, or panic on a concurrent map write.
- `Stocks.Candles` no longer sends a redundant `resolution` query parameter alongside the
  path segment; verified live that the API ignores it (a request to `/candles/D/` with
  `resolution=60` still returns daily candles).
- `earnings-analyzer` labelled the price reaction with the after-day candle's own
  intraday direction rather than the sign of the reaction it printed, producing rows like
  `+9.58% (bearish)`. Seven of twenty rows were wrong in a live run.

### Changed (2026-08-19, PR #33 review)
- Request construction is now a single serializer per endpoint, shared by the JSON method
  and both formatted facets (net −201 lines).
- The cancel-on-first-error concurrency policy (ADR-014) lives once in `internal/fanout`
  instead of being written out at four sites (net −102 lines); the entire suite passed
  without editing a single existing test.
- `Response.StatusError` replaces the same `200 OK`-but-not-`ok` guard repeated at 14
  call sites across four packages.
- The OpenAPI audit now fails on a stale allowlist entry. Its two allowlists are only
  consulted when the schema and the catalog disagree, so an entry that stopped being true
  was never read — which is how `stocks/earnings` `report` came to be recorded as
  unexposed one day before `WithEarningsReport` shipped. `report` is now in the catalog
  and covered by the reachability proof.

### Added (2026-08-05, CSV facet)
- `AsCSV()` on every service (`client.Stocks.AsCSV()`, `Options`, `Markets`, `Funds`) returns a facet that requests `format=csv` and hands back the API's raw CSV text (`response.CSVResponse.CSV()`) — not parsed into rows, matching the SDK's existing "typed JSON only" decoding decision (gap 40) and the Java SDK's own CSV facet (audited live for parity, see ADR-018). Endpoint coverage matches Java: Stocks (Candles, Quote, Quotes, Prices, News, Earnings — Candles auto-chunks and merges text exactly like the JSON path for >1yr intraday ranges), Options (Chain, Quote, Expirations, Quotes — returning `map[string]*response.CSVResponse`, one entry per symbol), Markets (Status), Funds (Candles). `Options.Lookup` and `Markets.StatusHistory` have no CSV facet, matching Java.
- `WithColumns`/`WithAddHeaders`/`WithHumanReadable` (ADR-013) now have a real effect for the first time — previously documented as "no effect on JSON, CSV-only" because there was no CSV request to apply them to.
- Closes the `IsCSV()` gap from the 2026-08-05 mutation-testing sweep (gap 42's deferred item, now gap 43): it was public API describing a capability — CSV support — that didn't exist.

### Fixed (2026-08-13, examples)
- **Seven command-line examples reported credits wrong.** They printed
  `client.RateLimits().Consumed` as a session total — `portfolio-monitor` called
  it "Session credits used" — but that field is the last completed response's
  own cost, never a running sum (requirements §8.4). They now report
  `Credits remaining` plus what the last request cost, and each says how to
  total a session properly (gap 56)

### Added (2026-08-13, examples)
- `examples/response-formats`: the `*Response` the other examples discard —
  typed JSON against the raw `AsCSV()` facet on the same endpoint, the
  `IsJSON`/`IsCSV`/`IsHTML` predicates, `Body()`, `SaveToFile()`, and the
  correct way to total session credits. Nothing in the tree exercised the CSV
  facet or `Body()` until now (gap 56)

### Added (2026-08-12, §13 test checklist closeout)
- `TestClient_RateLimitMetadataIsRequestScopedUnderConcurrency`: 40 concurrent
  quotes against a mock that answers each symbol with distinct rate-limit
  headers, asserting every response carries its own request's numbers rather
  than a shared client-level snapshot. This is the defect sdk-py and sdk-php
  both document as live in production; Go avoids it by construction, and this
  pins it. Closes the last unticked item of the requirements §13 unit-test
  checklist (gap 55)

### Changed (2026-08-12, reverse parity audit) — BREAKING
- `Options.Quotes`, `Options.QuotesBySymbol` and their `Get*` wrappers now take
  `[]string` plus variadic `QuoteOption` instead of variadic `...string`.
  The batch methods previously accepted no options at all, so a historical
  window was expressible for a single contract but not for several. The new
  shape also matches `stocks.Quotes`. Migration: wrap the symbols in a slice —
  `Quotes(ctx, a, b)` becomes `Quotes(ctx, []string{a, b})` (gap 54)
- `options.WithRange` now takes a typed `options.Moneyness` (`MoneynessITM`,
  `MoneynessOTM`, `MoneynessAll`, `MoneynessUnset`) instead of a raw string, so a
  typo cannot reach the wire as a silently-ignored filter (gap 54)

### Added (2026-08-12, reverse parity audit)
- `Options.LookupQuery` / `GetLookupQuery`: resolve an OCC symbol from a
  free-form human description such as `"AAPL 7/26/23 $200 Call"` — the input the
  endpoint exists to parse, and which the typed `Lookup` cannot express. `Lookup`
  now delegates to it (gap 54)
- `Response.Body()`: the raw response payload, returned as a copy. It was cached
  but unexported, reachable only through `SaveToFile` (gap 54)
- `Response.String()`: a one-line summary of status, no-data, body size and
  remaining credits, required by §11.6. It never includes the body itself (gap 54)
- `OptionQuote.Rho`: the API models rho internally but does not serialize it on
  the chain or quotes endpoints today, so the field is zero until it does (gap 54)
- `stocks.WithEarningsReport`: the `report` parameter, which the API declares but
  does not currently act on (gap 54)

### Added (2026-08-12, parameter-coverage audit)
- Two-layer audit that keeps the SDK's parameter catalog honest against the API's
  published OpenAPI schema, closing the blind spot that hid gaps 45, 46 and 50-52
  while every existing suite stayed green (gap 53):
  - `internal/apicatalog/openapi_test.go` compares the catalog against a
    checked-in copy of the schema and fails when either side knows a parameter
    the other does not, or when the API grows a path that is neither implemented
    nor declared out of scope. Runs offline in normal CI
  - `integration/openapi_drift_test.go` re-fetches the live schema and fails with
    an exact diff when the checked-in copy has gone stale. Rewrite it after review
    with `MARKETDATA_UPDATE_OPENAPI=1`

### Added (2026-08-11, API parameter parity)
- `stocks.WithCandle(bool)` / `stocks.WithQuotesCandle(bool)`: request the current
  session's OHLC alongside a quote. `stocks.Quote` gained `Open`, `High`, `Low`,
  and `Close`, which the API omits entirely unless the parameter is sent. sdk-java
  has had this on both its quote requests (gap 50)
- `stocks.WithCandleExchange(code)` / `stocks.WithCandleCountry(cc)`: resolve a
  candles symbol on a specific exchange or country, for tickers listed in more
  than one place (gap 51)
- `options.QuoteLastN(n)` / `options.QuoteLastNUntil(n, to)`: countback modes for
  the single-contract quote window, the last of the four date-window endpoints to
  lack them. A bare countback is anchored with `to=` today, because the endpoint
  ignores an unanchored one — the same defect and remedy as `stocks/earnings`
  (gap 52)
- `Options.QuoteHistory` (and `GetQuoteHistory`): returns every quote a historical
  window selects, in the API's order (gap 52)

### Fixed (2026-08-11, API parameter parity)
- **`Options.Quote` silently discarded rows.** A historical window selects one
  quote per day and only the first was returned — a caller asking for a week of
  quotes received one, with no error and no sign that the rest were dropped. This
  already affected `QuoteRange`. `Quote` still returns a single quote, as its name
  says, but its godoc now states the truncation and points at the new
  `QuoteHistory` for the full series (gap 52)

### Known API defects
- `options/quotes` ignores the `to` anchor when a `countback` is present and
  returns the contract's **oldest** n quotes rather than the n ending at `to`
  (verified live 2026-08-11). The SDK sends the parameters as documented and does
  not compensate; `integration/discrepancy_test.go` carries the strict assertion
  that will pass once the API is corrected

### Added (2026-08-11, API parity)
- `options.Strikes(...float64)` and `options.ByDeltas(...float64)`: the chain's
  `strike` and `delta` parameters both accept a comma-separated list, letting a
  spread or a strangle be priced in one request instead of one call per leg.
  A strike list was previously reachable only through the undocumented
  `StrikeExpr` escape hatch, and a delta list was not expressible at all (gap 46)
- `Options.QuotesBySymbol` (and `GetQuotesBySymbol`): returns one entry per
  requested option symbol, with a nil value where the API had no data. `Options.Quotes`
  omits those symbols, so its slice is shorter than the input with no indication of
  which contracts were dropped; `Quotes` is unchanged and its godoc now points here
  when that distinction matters (gap 47)
- `Stocks.BulkCandles` accepts an empty symbol list when combined with
  `WithSnapshot(true)`, which requests the API's market-wide snapshot — a candle
  for every symbol it covers. This was previously unreachable. The response is
  many thousands of rows and is billed accordingly (gap 48)

### Fixed (2026-08-11, API parity)
- `Options.Lookup` now validates all four of its arguments before the request. A
  zero expiration, a strike at or below zero, or an option type other than
  `options.Call`/`options.Put` was previously interpolated into the query
  verbatim, producing a request like `"AAPL 0001-01-01 0 "` whose 404 was
  reported as no-data — so a malformed call returned an empty string and a nil
  error, indistinguishable from a contract that does not exist. Such calls now
  return a `ValidationError` naming the field (gap 49)

### Fixed (2026-08-11, options chain expiration)
- **The full options chain is now reachable.** `options.AllExpirations()` is a new
  [`ExpiryFilter`] mode that sends `expiration=all`, returning every listed
  expiration. It was previously impossible to express: no combination of chain
  options produced that parameter, so callers could only ever get the front-month
  expiration. Measured live against production — an unfiltered AAPL chain returns
  190 contracts across 1 expiration, `AllExpirations()` returns 3528 across 24 —
  so expect a much larger response, billed accordingly (gap 45)
- `Options.Chain`'s godoc claimed that an unfiltered request "returns the full
  chain of all active contracts across every expiration". It returns only the
  front-month expiration. The documentation now states the real default and points
  at `AllExpirations()` for the whole chain

### Fixed (2026-08-05, T-4 wire-shape coverage)
- **`Utilities.Headers()` now actually works.** It previously decoded assuming a `{"headers": {...}}` envelope; the live API instead echoes the request's own headers as a flat top-level object with no wrapper key, so every call silently returned an empty map. Found by extending `integration/wire_shape_test.go`'s live wire-shape comparison from its original 6-endpoint sample to the full 14-endpoint set (gap 44) — the offline mock and fixture had both serialized the SDK's own (wrong) struct, so neither could ever have caught the mismatch
- `stocks.NewsArticle` gained an `Updated time.Time` field: `stocks/news` sends a response-level `updated` timestamp (one value for the whole list) that the SDK silently dropped
- `integration/wire_shape_test.go` now covers all 14 endpoints (`funds/candles`, `options/lookup`, `stocks/bulkcandles`, `stocks/earnings`, `stocks/news`, `stocks/prices`, `utilities/headers`, `utilities/user` added), closing T-4

### Fixed (2026-08-05)
- The §7 "client initialized" INFO log point now carries `base_url` and `api_version` — the 2026-08-04 fix added the log line itself but left it bare
- `docs/adr/DEVIATIONS.md`'s note on ADR examples assuming a 2-value method return now also lists ADR-004, ADR-012, and ADR-014, which each have their own such example beyond the ones already covered there for other reasons
- `Version()`'s godoc now says "unknown" (matching the code) instead of "dev" for a from-source build without stamped version info
- `WithAPIVersion` rejects values containing `/`, `\`, or `..` instead of interpolating them straight into the request path, closing a path-traversal-shaped URL from a mistyped option or `MARKETDATA_API_VERSION`
- The `.env` parser no longer loses a quoted value's closing quote when followed by an inline comment (`KEY="value" # comment` kept the quotes literally) and no longer silently drops `export<TAB>KEY=value` lines (only a literal space after `export` was recognized)
- `Options.Expirations` no longer returns `nil` for a 200 OK response with an empty expirations list — it now returns `&Expirations{}` with an empty `Dates` slice, matching the documented "nil only on 404" contract. Previously a caller trusting `Response.NoData` (as the docs instruct) over a manual nil check could dereference a nil pointer
- An explicitly-empty real environment variable (e.g. `MARKETDATA_TOKEN=`) is now honored instead of silently falling back to a `.env` file's value, restoring the common CI pattern of forcing demo mode with an empty override
- The rate limiter's own pre-flight rejection (the SDK catching a request before it was ever sent) now logs at WARN instead of ERROR — it's expected throttling, not a server-reported failure. A non-auth startup failure (network down, 5xx) no longer logs ERROR twice: the background rate-limit-priming goroutine that duplicated the synchronous validation call's log line is now silent, matching its already-documented "best-effort, error discarded" design
- `Close()` no longer calls `CloseIdleConnections()` on a caller-injected `WithHTTPClient` transport — that transport is intentionally shared with the caller (see the 2026-07 fix for gap 29), and closing its idle connections reached outside the SDK into the caller's own pooled connections

### Added (2026-07-28, spec conformance)
- The default logger now emits the canonical cross-SDK log format (`{timestamp} - marketdata.client - {LEVEL} - {message} key=value ...`, requirements §7) instead of slog's stock text format, and `WithDebug(true)` / `Client.Debug(true)` now actually raise the default logger to DEBUG at runtime (previously debug records were filtered unless `MARKETDATA_LOGGING_LEVEL=DEBUG` was also set). Loggers injected with `WithLogger` keep their own handler, format, and level
- `MARKETDATA_BASE_URL`, `MARKETDATA_API_VERSION`, and `MARKETDATA_MODE` environment variables are now honored (requirements §4); previously documented but not implemented. The full cascade is tested: `.env` < environment < client options
- CI now tests the declared minimum Go version (1.22) and current stable (requirements §13); the `toolchain` pin was removed from go.mod and the `go` directive relaxed to `1.22`
- CI runs on pushes and pull requests to the `development` integration branch
- `examples/README.md` index (what each example shows, SDK surface exercised, credit costs) linked from the README

### Changed
- `Quote`/`Price`/`Earning` percentage displays now render the API's fractional values ×100: `String()` shows a -0.21% day as `(-0.21%)` instead of `(-0.00%)`, `Earning.String()` renders the earnings surprise as a percentage instead of $-formatting the raw fraction, and the example programs' CHG%/surprise columns were fixed the same way. The field values themselves are untouched and their docs now state they carry the wire fraction (-0.0021 means -0.21%), matching the Java SDK's modeling
- `MarketStatus.IsClosed()` now compares the API's status string ("closed") instead of negating `Open`, so a day outside the holiday calendar's coverage (empty status) reads as neither open nor closed rather than falsely "closed"; `MarketStatus.IsEarlyClose()` was removed — the public API's status vocabulary is strictly "open"/"closed" (early-close days report as "open", verified live), so the method could never return true, and the Java SDK models no such state either
- `NotFoundError`/`ErrNotFound` docs now state plainly that the SDK never returns them: every 404 is "no data" (`Response.NoData`), by design; the type remains defined for the cross-SDK error taxonomy required by requirements §6.1, matching the Java SDK
- `Options.Expirations` now returns `*options.Expirations` (fields `Dates []time.Time` and `Updated time.Time`) instead of a bare `[]time.Time`: the API sends a response-level `updated` timestamp on this endpoint and the SDK silently discarded it. `Updated` is the zero time when the API omits the field. `OptionsChain.Updated` was removed for the inverse reason — the chain endpoint has no chain-level timestamp in its schema (each contract's `OptionQuote.Updated` is the wire truth), so the field was never populated and always read as the zero time. Both types now mirror the API response exactly, matching the Java SDK's modeling
- `Stocks.Quotes` (bulk) now takes its own `QuotesOption` type and no longer accepts `WithFiftyTwoWeek`/`WithExtended`: the API honors the `52week` parameter only for single-symbol requests, so passing it to a multi-symbol call silently returned zero values. 52-week data is now only expressible on `Stocks.Quote`, where it works; extended-hours on bulk quotes moved to the new `WithQuotesExtended` option. Enforced by a new negative-compile case

### Fixed
- Null or absent timestamps now decode to the zero `time.Time` (so `IsZero()` detects missing data) instead of rendering as the Unix epoch `1969-12-31 19:00 EST`; applies to every timestamp field across stocks, options, funds, markets, and utilities
- US/Eastern time is now correct on hosts without a system timezone database (scratch/alpine containers): the IANA database is embedded via `time/tzdata` (~450KB), replacing a fallback that pinned EST year-round and was an hour off during DST
- The example programs and godoc examples now guard the NoData path: several dereferenced a nil result after a 404 ("no data") response — `basic` printed quote fields, portfolio-monitor and watchlist-alerter called methods on a nil `MarketStatus`, and the stocks/markets/utilities/options examples skipped the nil check their own method docs describe
- Loading a `.env` file no longer mutates the process environment: values are parsed into the SDK's own configuration cascade (real environment variables still win) instead of being `os.Setenv`'d process-wide — previously every key in the file, MarketData-related or not, leaked to the whole host program depending on the working directory
- The `.env` parser handles the common dialect correctly: one balanced pair of surrounding quotes is stripped (inner quotes preserved — `KEY="say 'hi'"` no longer loses its closing quote), `export KEY=value` lines work, unquoted inline ` #` comments are cut, and a >64KiB line no longer silently discards the rest of the file (1 MiB buffer; beyond that the entries parsed so far are kept and an error is reported)
- An injected `WithHTTPClient` client is no longer mutated: the SDK now operates on a shallow copy (sharing the caller's `Transport`) instead of overwriting the caller's `Timeout` and `CheckRedirect` — which was both a surprise side effect and a data race with in-flight requests (it flaked CI on the PR #14 merge). A caller-supplied transport keeps its own dial settings; the SDK's 2s connect timeout applies only to the transport it builds itself
- The offline gate now actually works during an outage: a failed status probe counts as offline (previously errors were discarded and a downed host could never flip the optimistic default, so retries were never suppressed), and the probe bypasses the concurrency pool, retry loop, and rate-limit accounting so it stays responsive exactly when the pool is saturated
- Server error messages truncate on a UTF-8 rune boundary instead of mid-rune at byte 500
- Client-default universal parameters no longer collapse multi-value defaults to their last value, and the merge no longer writes into the request's own `url.Values`
- `Client.Debug(true)` at runtime now actually enables HTTP request logging: the internal HTTP client used to copy the debug flag once at construction, so toggling debug on a client built without `WithDebug` raised the logger level but never emitted the per-request records. The runtime debug state now lives in the internal client as an atomic (`SetDebug`), which also removes a data race when `Debug` was toggled concurrently with in-flight requests
- `Stocks.Earnings` with `LastN(n)` now actually returns the last n reports: the API silently ignores a `countback` that arrives without `to=` (its undocumented bare default is the upcoming-only window — API defect MarketData-App/api#283), so an unanchored `LastN(n)` returned the next scheduled unreported row, or NoData when a symbol sat in its post-earnings calendar gap. The SDK now anchors a bare earnings countback with an explicit `to=` of today (Eastern), which is honored today and remains semantically identical once the API defect is fixed. Candles and News keep sending bare countback, which their endpoints honor
- A true $0.00 EPS no longer collapses to `nil` in `Stocks.Earnings`: the wire structs decoded the four EPS arrays as `[]float64` (JSON null → 0) and only set the public `*float64` when the value was non-zero, so a company meeting its estimate exactly (`surpriseEPS: 0`) was indistinguishable from "not yet reported" and printed as `n/a` — the exact ambiguity the pointer fields are documented to prevent. The wire now decodes as `[]*float64`, preserving null vs zero end to end
- Intraday candle auto-splitting no longer duplicates chunk-boundary days. The API treats date-only `from`/`to` as inclusive on both ends, but consecutive year-sized chunks shared their boundary day, so every internal boundary's candles appeared twice in the merged result (~390 duplicated rows per boundary at 1-minute resolution, silently double-counting volume sums and backtests). Chunks are now disjoint (the next chunk starts the day after the previous one ends), the merge additionally dedupes identical adjacent timestamps as defense in depth, and the merged `Response` now reflects the last chunk that carried data, so a trailing no-data chunk can no longer mark a result that has candles as `NoData`. Found during the 2026-07-31 deep review; the same bug was confirmed in the Java, Python, and JS SDKs (issues filed: sdk-java#53, sdk-py#51, sdk-js#74)
- `NewClient` now propagates demo mode to the internal HTTP client. Previously the flag was never wired through, so demo (tokenless) clients sent a malformed empty `Authorization: Bearer ` header on every request, and the demo exemption from the rate-limit pre-flight was unreachable from the public constructor (demo mode only kept working because of the tracker's independent `limit=0` rule). An end-to-end test now exercises the demo path through `NewClient` itself
- Restored the 100% unit-test coverage gate in CI (was temporarily 98%): added tests for every previously uncovered branch, made `detectVersion` and the timezone loader testable via small refactors, removed the status cache's dead placeholder fetcher, and fixed a copy-paste bug where `TestQuote_EmptySymbol` exercised `Quotes` instead of `Quote`
- Context cancellation and deadline errors now surface uniformly as `*NetworkError` (with `Timeout` set for deadlines and the context error reachable via `errors.Is`) regardless of where the context fires — waiting for a pool slot, sleeping between retries, or mid-request. Previously the pool-wait and backoff paths returned the bare context error while the transport path wrapped it
- Demo mode no longer wedges after the first request: the API marks anonymous responses with `limit=0`/`remaining=0` rate-limit headers, which the pre-flight check previously recorded and then rejected every subsequent call with a spurious `RateLimitError`. Demo mode now skips the pre-flight reservation, and a `limit=0` snapshot is treated as unmetered access rather than exhaustion (matching the Java SDK)

### Security
- Caller-supplied symbols are percent-encoded before URL path interpolation, preventing path-segment injection (e.g. `AAPL/../../user`)
- Server-supplied error messages are truncated (500 chars) and stripped of control characters before embedding in errors, preventing log forging and terminal escape spoofing
- Server-supplied Retry-After delays are capped at 10 minutes; larger or overflowing values fall back to calculated backoff
- Tokens containing non-printable or non-ASCII characters and malformed base URLs are rejected at client construction
- Debug logs redact query strings; full URLs appear only in error support context

### Removed
- Dead exports that no SDK path used: the `UniversalParams` struct (universal parameters are client options), `stocks.formatInt`, and the internal `Tracker.Exceeded`
- The v1 strikes endpoint has no v2 equivalent (dropped from the required SDK surface); use Chain with strike filters instead

[Unreleased]: https://github.com/MarketDataApp/sdk-go/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/MarketDataApp/sdk-go/releases/tag/v2.0.0
