# ADR-013: Universal Parameters

## Status
Accepted (revised 2026-07-11 during the compile-time-safety hardening).

## Context

The MarketData API supports four universal parameters that apply to every
endpoint: `dateformat`, `columns`, `headers`, and `human`. They control the
wire representation of the response. The SDK, however, decodes responses into
strongly-typed Go structs, and three of these parameters change the wire format
in ways that are incompatible with that typed decoding:

- `dateformat` changes how dates are encoded. The SDK's response types expect
  the API's default numeric (unix) encoding; `spreadsheet`/`timestamp` would
  break `time.Time` decoding.
- `human` changes field names and value formatting (e.g. numbers as formatted
  strings), which breaks decoding into `float64`/`int64` fields.
- `headers` controls whether a header row is emitted in **CSV** output. The SDK
  always requests JSON, so it has no effect on decoded responses.

Only `columns` is safe with typed decoding: filtering to a subset of columns
merely leaves the omitted fields at their Go zero values.

## Decision

Universal parameters are exposed as **client-level functional options** that
populate the request's default query parameters (participating in the
configuration cascade of ADR-012). They are not per-method options: a single
option value cannot satisfy every resource's distinct, sealed option interface
without reintroducing the very cross-wiring the hardening removed, and per-call
formatting overrides are rarely needed.

```go
client, _ := marketdata.NewClient(
    marketdata.WithToken(tok),
    marketdata.WithColumns("last", "bid", "ask"), // safe: reduces payload
)
```

| Option                       | Param        | Safety with typed decoding |
|------------------------------|--------------|----------------------------|
| `WithColumns(cols...)`       | `columns`    | Safe (omitted fields → zero values) |
| `WithMode(Mode)`             | `mode`       | Safe. `live`/`cached`/`delayed` (premium). Cache miss → 204 (see below) |
| `WithMaxAge(string)`         | `maxage`     | Safe. Max cached-data age with `mode=cached` |
| `WithLimit(int)`             | `limit`      | Safe. Cap results / override endpoint default |
| `WithOffset(int)`            | `offset`     | Safe. Pagination offset (with `limit`) |
| `WithDateFormat(fmt)`        | `dateformat` | **Advanced** — can break typed date decoding |
| `WithHumanReadable(bool)`    | `human`      | **Advanced** — incompatible with typed decoding |
| `WithAddHeaders(bool)`       | `headers`    | No effect on JSON (CSV-only) |

The parameter names track the official docs — `mode` (not the older `feed`).
`WithMode(ModeCached)` can produce an **HTTP 204** (cache miss); the SDK treats
204, like 404, as a no-data response (`response.IsNoData`), so it never surfaces
as a decode error. `mode`/`limit`/`offset`/`maxage` are genuinely per-request in
spirit but are exposed as client-level defaults for the reasons below; callers
that need different values per request use separate clients (documented in
docs/RESIDUALS.md).

Each option's godoc states any caveat plainly. Environment variables
(`MARKETDATA_DATE_FORMAT`, `MARKETDATA_COLUMNS`, `MARKETDATA_USE_HUMAN_READABLE`,
`MARKETDATA_ADD_HEADERS`) remain a lower-priority source in the cascade.

### Reachability and residuals

All four universal parameters are **reachable** from idiomatic Go (the options
above) and from the environment. The SDK also sets `dateformat` internally at
the method level where a specific format is required for correct decoding (for
example the options-expirations endpoint requests `unix`); method-level values
take precedence over the client default, so the SDK's own correctness is
preserved regardless of the client default.

`dateformat`, `human`, and `headers` are **documented residuals** with respect
to typed decoding: they are reachable, but using them to alter the response
format is at the caller's own risk and is intended for callers that consume the
raw response themselves. This is a deliberate design choice, not an omission.

## Consequences

### Positive
- Every universal parameter is reachable via idiomatic Go, not only via env vars.
- `columns` gives a safe, useful payload-reduction knob.
- Typed decoding stays correct by default; the SDK owns `dateformat` where it
  must.

### Negative
- Universal parameters are client-level, not per-request. A caller wanting a
  per-request `columns` filter must create a second client. (Accepted: this is
  rare and keeps the option surface consistent with the sealed per-method
  option families.)

### Trade-offs vs. the original design
The original ADR proposed a shared `UniversalOption` accepted by every method.
That was never implemented and conflicts with the hardened design, where each
method's option interface is deliberately distinct so that method-specific and
mutually-exclusive parameters cannot be mixed. Client-level options achieve
reachability without weakening that guarantee.

## References
- Requirements: Section 3 (Universal Parameters)
- Related: ADR-004 (request/response pattern), ADR-012 (configuration cascade)
