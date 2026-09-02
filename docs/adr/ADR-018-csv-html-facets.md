# ADR-018: CSV and HTML Response Facets

## Status
Accepted (2026-08-05).

## Context

`response.Response.IsCSV()`/`IsHTML()` were public content-type checks with
nothing behind them: the SDK always requests `format=json`, so `IsCSV()`
could only ever be false, and there was no path to request CSV or HTML at
all (gap 40 / ADR-013 recorded this as a deliberate scope decision — "Go v2
decodes JSON into typed structs only"). The mutation-testing sweep that
found `IsCSV()` misleading (2026-08-05 gap 42) flagged it for a real fix
rather than removal.

The Java SDK (`sdk-java`, audited live at `main`@`25e95a9`) has already
solved this and was used as the reference design:

- **CSV** is a real, functional, public facet: `client.stocks().asCsv()...`
  requests `?format=csv` and an `Accept: text/csv` header, and returns the
  raw response text (`CsvResponse.csv()`) — not parsed into rows. `isCsv()`
  on the base response interface is the same content-type check Go already
  had.
- **HTML** is built but deliberately not exposed: the wire plumbing
  (`Format.HTML`, `HtmlResponse`, a full `*HtmlResource` per service
  mirroring the CSV facet) exists and is unit-tested, but the entry point
  (`asHtml()`) is package-private, with an explicit comment to flip it
  public "when the server supports format=html". `isHtml()` stays public
  today for detecting a misrouted request hitting the web tier, not a real
  `format=html` response.

Verified live against the production API before designing the Go facets
(not assumed from Java's behavior):

- `format=html` on a data endpoint (`stocks/quotes`) returns `HTTP 404` with
  a JSON error body — HTML is genuinely unsupported today, confirming the
  Java team's "built but inert" framing is still accurate.
- **Error bodies are serialized in the requested format, not always JSON.**
  A 400 from `?format=csv` comes back as `s,errmsg\nerror,"Bad
  parameters..."` — CSV-shaped, not a JSON object. Naively reusing the
  existing JSON error parser on this body silently degrades every CSV/HTML
  error to a generic HTTP status message, discarding the API's actual
  `errmsg`.
- **The candles endpoint's "no data" signal is inconsistent between
  formats.** The same out-of-range request that returns `HTTP 404` with
  `{"s":"no_data",...}` for JSON returns `HTTP 200` with a degenerate body
  (`0\n""`) for `format=csv`. This is API behavior, not an SDK choice, and
  it means the JSON path's `NoData` contract cannot be replicated
  faithfully for CSV/HTML without inventing a sentinel-body-sniffing rule
  the API does not document or guarantee.

## Decision

Mirror Java's two-tier design, adapted to Go's `(T, *Response, error)` /
functional-options idiom, with two Go-specific resolutions for the API
inconsistencies above (confirmed with the maintainer before implementation):

### CSV: real, public, per-service facet

Each service gains an `AsCSV() *CSVService` method (`stocks.Service`,
`options.Service`, `markets.Service`, `funds.Service`) returning a facet
with one method per endpoint that has a CSV counterpart in Java — the same
endpoint set, not invented: Stocks (Candles, Quote, Quotes, Prices, News,
Earnings), Options (Chain, Quote, Expirations, Quotes), Markets (Status),
Funds (Candles). `Options.Lookup` and `Markets.StatusHistory` have no CSV
facet, matching Java exactly.

Each method returns `(*response.CSVResponse, error)` — no separate
`*response.Response`, since `CSVResponse` embeds `Response` and adds
`CSV() string` for the raw text:

```go
csv, err := client.Stocks.AsCSV().Quote(ctx, "AAPL")
if err != nil {
    log.Fatal(err)
}
fmt.Print(csv.CSV())
```

The SDK does not parse the CSV into rows — same as Java's `CsvResponse`.
Parsing would mean maintaining a second decoder per endpoint alongside the
JSON one for no behavioral gain: everything the typed JSON methods already
expose is available there, structured.

`options.Quotes`' CSV facet fans out one request per symbol like its JSON
counterpart, but returns `map[string]*response.CSVResponse` instead of a
merged slice — matching Java's `Map<String, CsvResponse>` shape, since each
contract's CSV text is independent (unlike candle chunks of the same
series, there's nothing to concatenate).

`stocks.Candles`' CSV facet applies the identical >1-year intraday
chunking decision as the JSON path (same `candleChunks` helper, extracted
from `candlesSplit` for reuse — see `marketdata/stocks/chunks.go`), fetches
chunks concurrently with the same first-error-cancels-siblings behavior
(ADR-014), and merges the CSV text in chunk order instead of merging typed
candles — dropping the repeated header line from every chunk after the
first, detected structurally (comparing each chunk's leading line to the
first chunk's) rather than by threading through whether the `headers`
universal param was set.

### HTML: built, unexported, ready to flip on

Each service also gets an `asHTML() *htmlService` — lowercase, unreachable
from outside the package — mirroring the CSV facet method-for-method.
`response.HTMLResponse` (with `HTML() string`) and `response.NewHTML` exist
alongside `CSVResponse`/`NewCSV`. This is Java's exact pattern: the type and
request plumbing exist and are exercised by tests, but the public entry
point stays off until `format=html` actually serves data. Flipping it on
later is renaming `asHTML`/`htmlService` to `AsHTML`/`HTMLService` (and,
per Go convention, likely dropping the `as` prefix to match `AsCSV`) — no
new design.

`response.Response.IsHTML()` (already present, unchanged by this ADR)
keeps working exactly as before: a public, functional content-type check,
independent of whether the facet is exposed — same relationship Java has
between `isHtml()` (public) and `asHtml()` (package-private).

### Resolving the two API inconsistencies

- **CSV-shaped error bodies**: `internal/http.parseAPIError` now branches on
  the response's Content-Type. A `text/csv` error body is read with
  `encoding/csv` (header row + one data row, mapped by column name into the
  same fields the JSON path populates) before falling into the existing
  status-code-to-error-type switch, so `RateLimitError`, `BadRequestError`,
  etc. carry the real `errmsg` regardless of which facet produced them.
  Malformed or short CSV degrades to the same generic-status-text fallback
  the JSON path already had for unparseable bodies — no new failure mode.
- **NoData inconsistency**: resolved by *not* replicating `NoData` for
  CSV/HTML at all. `CSVResponse`/`HTMLResponse` have no `NoData` field;
  `internal/http.Client.GetFormatted` treats 404/204 as success (not an
  error) the same way `Get` does, but returns the raw body exactly as the
  API sent it rather than synthesizing a no-data marker. A caller gets
  whatever text the API returns for a given status/format combination,
  including the `0\n""` degenerate candles body — documented here rather
  than silently masked or guessed at.

### Alternatives considered

- **Parse CSV into structured rows.** Rejected: doubles the decode surface
  per endpoint for a facet whose entire purpose (per Java's own design and
  the RESIDUALS.md CSV decision) is to hand back what the API sent, not to
  duplicate the typed JSON path.
- **Skip the CSV-error-body fix and accept degraded error messages for
  CSV/HTML** (Java's own approach — `HttpStatusMapper` never reads the
  body at all, using fixed per-status messages). Considered and rejected
  for Go specifically: the existing JSON path already extracts and embeds
  `errmsg`, and silently degrading error quality only for these two facets
  would be a regression a caller could hit without any signal.
- **Replicate NoData by sniffing the `0\n""` candles body.** Rejected as
  fragile: that exact body is observed, not documented or contractually
  guaranteed by the API, and sniffing it would silently break if the API
  ever changes the shape without notice — worse than a caller seeing the
  raw text and deciding for themselves.

## Consequences

- `IsCSV()` is no longer public API describing a capability that doesn't
  exist: `AsCSV()` on every service backs it with a real, working facet.
- `WithColumns`/`WithAddHeaders`/`WithHumanReadable` (ADR-013, previously
  "no effect on JSON, CSV-only" — inert because there was no CSV) now have
  a real effect for the first time, with no change to their own API.
- HTML support ships as inert, tested, close-to-zero-risk surface — no new
  design work needed if/when the API adds `format=html` for data endpoints.
- Every service package gains a small, mechanical duplication of each
  endpoint's parameter-building logic (one `xPath` function per endpoint,
  shared between the CSV and HTML facet methods) rather than refactoring
  the already-shipped, well-tested JSON methods to share it — deliberately
  chosen to keep this change additive-only and avoid risking regressions in
  existing, stable code paths. The one exception is `stocks.candleChunks`,
  extracted from `candlesSplit` because the year-chunking boundary logic is
  genuinely correctness-critical (gap 24) and must not exist as two
  independently-maintained copies.
