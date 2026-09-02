# ADR-017: Compile-Time Parameter Exclusivity

## Status
Accepted (2026-07-11).

## Context

Several Market Data endpoints accept parameters that are mutually exclusive. The
v1-style surface resolved these at *runtime* by silent precedence, which the live
API itself handles inconsistently (verified 2026-07-11):

- stocks candles `from`+`to`+`countback` → the API silently drops `countback`.
- options chain `expiration`+`dte` → the API silently honors `expiration`.
- options chain `expiration`+`month`+`year` → the API returns 404 (empty).
- options quote `date`+`from`+`to` → the API returns HTTP 400.
- options chain strike selection (`strike`) vs `delta` → the API silently honors `strike`.

Three different failure modes for the same class of mistake. A published SDK
embedded in third-party apps must not let a caller *write* these combinations.

## Decision

Make incompatible parameters **unrepresentable in the type system**. Every
mutually-exclusive group collapses into a single value; the caller can only ever
supply one mode.

### Mechanism: sealed union values carried by one functional option

The SDK keeps its idiomatic functional-options surface (consistent with
`grpc-go`; the service/`ctx`-first/`(T, *Response, error)` shape follows
`google/go-github`). Each exclusivity group becomes ONE option whose argument is a
**sealed union interface** — an interface with an unexported method, so only the
package's own constructors can produce a value:

```go
// A sealed union: third-party code cannot implement window() and so cannot
// construct an illegal or arbitrary date mode.
type DateWindow interface{ window() params.Window }

func OnDate(d time.Time) DateWindow           // date=
func Between(from, to time.Time) DateWindow    // from=&to=
func Since(from time.Time) DateWindow          // from=
func Until(to time.Time) DateWindow            // to=
func LastN(n int) DateWindow                   // countback=
func LastNUntil(n int, to time.Time) DateWindow // countback=&to=

func WithCandleWindow(w DateWindow) CandleOption
```

Because `WithFrom`/`WithTo`/`WithCountback` no longer exist, the old illegal call
does not compile (undefined symbol), and the union makes the modes one value.

The unions:

| Group | Union | Constructors |
|---|---|---|
| stocks/funds date window | `DateWindow` | `OnDate` `Between` `Since` `Until` `LastN` `LastNUntil` |
| options strike/delta | `StrikeFilter` | `Strike` `StrikeRange` `MinStrike` `MaxStrike` `StrikeExpr` `ByDelta` |
| options expiry | `ExpiryFilter` | `OnExpiration` `InDTE` `InMonth` `InYear` `InMonthOfYear` `ExpirationBetween` |
| options expiration-type include/exclude | `ExpirationTypeFilter` | `IncludeExpirationTypes` `ExcludeExpirationTypes` |
| options quote historical | `OptionQuoteWindow` | `QuoteOnDate` `QuoteRange` |

The options-chain expiration selector includes `ExpirationBetween(from, to)`:
per the API docs (confirmed live), a chain `from`/`to` range filters
*expirations* and is mutually exclusive with `dte`, so it is an `ExpiryFilter`
mode rather than a historical range. The chain's historical "as of" selector is
a single day exposed as the independent `WithChainDate` option (a chain snapshot
is taken as of one date; it is not a union).

The expiration-type filter is a *value-dependent* cross-field constraint rather
than a classic pick-one group: the API allows any set of `weekly`/`monthly`/
`quarterly` to be all-included (`true`) or all-excluded (`false`), but forbids
mixing inclusion and exclusion in one request. Independent `bool` options would
let `WithWeekly(true), WithMonthly(false)` compile and fail only at runtime, so
the three flags collapse into one `ExpirationTypeFilter` that is *either*
`IncludeExpirationTypes(...)` or `ExcludeExpirationTypes(...)` — the include/
exclude choice is a single value, so the illegal mix is unrepresentable. (This
was found by the adversarial grader and fixed rather than documented as a
residual, keeping the "no exclusivity residuals" guarantee true.)

Markets is the one case where the exclusivity is between *methods* (single-date
`Status` vs ranged `StatusHistory`); there the fix is two distinct sealed option
types (`StatusOption`, `HistoryOption`) with a shared `WithCountry` that satisfies
both, so `WithDate` and `WithHistoryWindow` are method-exclusive at compile time.

### Alternatives considered

- **Request structs** (`ChainRequest{Strike: …, Expiry: …}`): even stricter (a
  field holds one value, so double-set is impossible), and idiomatic per
  go-github's options structs. Rejected as the default because it is a larger
  break from the SDK's established functional-options style and churns every call
  site/example/TUI more; the union *values* are identical, so a later switch is
  mechanical.
- **Separate method variants** (`Candles`, `CandlesCountback`, …): combinatorial
  explosion for chain (three independent groups). Rejected.

### Enforcement

- `internal/negcompile` compiles a corpus of illegal snippets in an isolated
  module and asserts each **fails to build** with the expected diagnostic; a
  positive corpus asserts the legal combinations build. Run in CI.
- Value-range rules that types cannot express are validated pre-network (see
  `docs/RESIDUALS.md`).

## Consequences

- Illegal parameter combinations cannot be written; the compiler is the
  enforcement. No silent precedence, no malformed request ever sent.
- The date window also reached a previously-**unreachable** parameter: stocks and
  funds candles now expose the API's single-day `date=` via `OnDate`.
- Breaking change to the public surface (v2 is unreleased, so this is free).
  Examples, TUIs, and ADRs updated to match.
- Slightly more verbose call sites (`WithCandleWindow(stocks.Between(a, b))`), in
  exchange for illegal calls being impossible.
