# Residuals — what is enforced at runtime rather than at compile time

The hardening goal is that incompatible parameters are **unrepresentable** (do not
compile). This document lists, per the house rules, every case that is *not* a
compile-time guarantee, with its justification. Read it as the exhaustive answer
to "what did you have to fall back to runtime for?"

## Headline result: zero exclusivity residuals

Every mutually-exclusive parameter group in the API is now compile-time
exclusive — there is no illegal *combination of different parameters* that can be
written and rejected only at runtime. Each group collapsed into a single
sealed-union value carried by one option (or, for markets, into two distinct
method-scoped option types):

| Exclusivity group | Mechanism (unrepresentable in combination) |
|---|---|
| candles/earnings/news `date`/`from`/`to`/`countback` (stocks, funds) | `DateWindow` sealed union (`OnDate`/`Between`/`Since`/`Until`/`LastN`/`LastNUntil`) |
| options chain strike vs delta (`strike`/`delta`) | `StrikeFilter` union (`Strike`/`Strikes`/`StrikeRange`/`MinStrike`/`MaxStrike`/`StrikeExpr`/`ByDelta`/`ByDeltas`) |
| options chain expiry (`expiration`/`dte`/`month`/`year`/`from`+`to`) | `ExpiryFilter` union (`AllExpirations`/`OnExpiration`/`InDTE`/`InMonth`/`InYear`/`InMonthOfYear`/`ExpirationBetween`) |
| options chain expiration-type include vs exclude (`weekly`/`monthly`/`quarterly`) | `ExpirationTypeFilter` union (`IncludeExpirationTypes`/`ExcludeExpirationTypes`) — the API forbids mixing include (`true`) and exclude (`false`), and the union makes the mix unrepresentable |
| options quote historical (`date` vs `from`/`to` vs `countback`) | `OptionQuoteWindow` union (`QuoteOnDate`/`QuoteRange`/`QuoteLastN`/`QuoteLastNUntil`) |
| markets `Status` (single date) vs `StatusHistory` (range) | distinct `StatusOption` / `HistoryOption` types; `WithDate` and `WithHistoryWindow` are method-exclusive |

The negative-compile suite (`internal/negcompile`) proves each illegal combination
fails to build; the positive suite proves the legal ones compile.

## Residual 1 — value-range validations (single-parameter, not exclusivity)

Some *values* for a single parameter cannot be constrained by Go's type system (an
`int` can hold `13`; a `float64` can hold `1.5`). These are validated **before any
network call** and return a `*sdkerrors.ValidationError`. They are not
incompatible-parameter combinations; they are invalid values for one parameter.

| Parameter | Rule | Where |
|---|---|---|
| countback | `> 0` | `internal/params.Window.Validate` |
| date window | present dates non-zero; `from <= to` | `internal/params.Window.Validate` |
| options `month` | `1..12` | `options` validation |
| options `year` | `1900..9999` | `options` validation |
| options `dte` | `>= 0` | `options` validation |
| options strike / price bounds | `> 0`; `StrikeRange` low `<=` high; `Strikes` non-empty with every element `> 0` | `options` validation |
| options `delta` | non-zero, `-1 <= delta <= 1` (negative selects puts); `ByDeltas` non-empty with the same rule per element | `options` validation |
| options `Lookup` arguments | expiration non-zero; strike `> 0`; type is `Call` or `Put` | `options` validation |
| options quote `countback` | `> 0`; `QuoteLastNUntil` also requires a non-zero `to` | `options` validation |
| options `LookupQuery` | non-blank query string | `options` validation |
| stocks `BulkCandles` symbols | non-empty, unless `WithSnapshot(true)` requests the market-wide snapshot | `stocks` validation |

Justification: types cannot express numeric ranges; pre-network validation is the
correct place (never send a knowingly-malformed request), and each rule has a test
proving it rejects before any network call. This is the documented exception the
house rules permit, applied only to values, never to combinations.

## Residual 2 — same union option passed twice is last-wins

Passing the *same* option twice (e.g. `WithStrike(Strike(150))` then
`WithStrike(StrikeRange(140,160))`) resolves to the last value, the standard
functional-options semantics (like calling `WithTimeout` twice). This is **not** a
combination of two incompatible parameters — it is one logical parameter set
twice — so it is intentionally allowed and not treated as an error. The
incompatible-parameter footguns (two *different* parameters) are unrepresentable.

## Residual 3 — universal formatting parameters are SDK-owned

`dateformat`, `human`, and `headers` are reachable (client options
`WithDateFormat`/`WithHumanReadable`/`WithAddHeaders` and environment variables)
but are documented residuals with respect to the SDK's typed decoding:

- `dateformat` — the SDK decodes the API's default numeric date encoding; the SDK
  sets it per-endpoint where a specific format is required. Overriding it globally
  can break typed date decoding.
- `human` — human-readable output changes field names/formats and is incompatible
  with typed decoding.
- `headers` — CSV-only; no effect on the JSON the SDK requests.

`columns` is safe (omitted fields decode as zero values) and fully supported via
`WithColumns`. See ADR-013.

## Residual 3b — universal parameters are client-level, not per-request

The universal parameters `mode`, `maxage`, `limit`, `offset`, and `columns` are
reachable only as **client-level** options (`WithMode`/`WithMaxAge`/`WithLimit`/
`WithOffset`/`WithColumns`), not per-call. The hardened design gives each method
its own sealed option type so method-specific and mutually-exclusive parameters
cannot be mixed; a single universal option that satisfied every one of those
distinct interfaces would reintroduce that cross-wiring, and per-request universal
options would require threading a universal holder through every method. A caller
that needs, say, `mode=cached` for bulk quotes and `mode=live` for a specific
quote uses two clients. This is a deliberate trade-off (reachability + the
compile-time guarantee) over per-request universal ergonomics — justified, not an
omission.

## Residual 4 — request-size (oversized symbol) is not capped by the SDK

An extremely long caller-supplied symbol produces a long URL that the API rejects
(4xx). The SDK does not cap request/symbol length: it is not a DoS against the
caller (that is the *response* side, which is capped — see ADR-011), and silently
truncating a symbol would be worse than a clear API rejection. Untrusted symbols
are always safely percent-encoded first (no injection); see
`internal/http/untrusted_input_test.go`.
