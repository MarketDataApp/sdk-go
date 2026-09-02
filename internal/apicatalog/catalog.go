// Package apicatalog is the authoritative, live-probed catalog of every
// parameter each Market Data API endpoint accepts, together with how the SDK
// reaches it. It exists to make "every supported parameter is reachable" a
// machine-checked property rather than an assertion: the companion
// reachability test drives the real public SDK surface and fails if any
// catalogued Query/Path parameter is not actually emitted on the wire.
//
// The catalog is derived from live probing of the API on 2026-07-11 (the live
// API is the source of truth, over the docs). Each parameter records its
// exclusivity Group: parameters sharing a non-empty group are mutually
// exclusive and, in the redesigned surface, are unrepresentable in
// combination (they collapse into a single sealed-union option).
package apicatalog

// Kind classifies how a parameter reaches the wire.
type Kind int

const (
	// Query is a URL query parameter the SDK emits from an option/argument.
	Query Kind = iota
	// Path is a URL path segment (a required positional argument).
	Path
	// Residual is a parameter that is reachable but SDK-owned or otherwise not
	// a plain user-facing option (documented, not required to be emitted by a
	// user call). See Notes for the justification.
	Residual
)

func (k Kind) String() string {
	switch k {
	case Query:
		return "query"
	case Path:
		return "path"
	case Residual:
		return "residual"
	default:
		return "unknown"
	}
}

// Param is one parameter accepted by one endpoint.
type Param struct {
	Endpoint string // logical endpoint, e.g. "stocks/candles"
	Name     string // wire name (query key) or path role
	Type     string // date | int | float | bool | string | enum | []string
	Group    string // exclusivity group; "" means independent/free
	Accepted string // accepted values / notes from live probing
	SDKPath  string // idiomatic SDK expression that emits it
	Kind     Kind
}

// All returns the full catalog.
func All() []Param {
	var all []Param
	all = append(all, stocksParams()...)
	all = append(all, fundsParams()...)
	all = append(all, optionsParams()...)
	all = append(all, marketsParams()...)
	all = append(all, utilitiesParams()...)
	all = append(all, universalParams()...)
	return all
}

func stocksParams() []Param {
	const q = Query
	return []Param{
		// stocks/quotes (served by bulkquotes). The API honors 52week only
		// when exactly one symbol is sent (multi-symbol requests silently
		// ignore it), so the SDK exposes it on Quote — never on Quotes.
		{"stocks/quotes", "symbols", "[]string", "", "comma-separated tickers", "Stocks.Quotes(ctx, symbols)", q},
		{"stocks/quotes", "52week", "bool", "", "52-week hi/lo; single-symbol only, so Quote-only in the SDK", "Stocks.Quote(ctx, symbol, stocks.WithFiftyTwoWeek(true))", q},
		{"stocks/quotes", "extended", "bool", "", "extended-hours data", "stocks.WithExtended(true) / stocks.WithQuotesExtended(true)", q},
		{"stocks/quotes", "candle", "bool", "", "adds session OHLC to the quote", "stocks.WithCandle(true) / stocks.WithQuotesCandle(true)", q},
		// stocks/candles
		{"stocks/candles", "symbol", "string", "", "underlying ticker", "Stocks.Candles(ctx, symbol)", Path},
		{"stocks/candles", "resolution", "enum", "", "1..240(min/hour),D,W,M,Y", "stocks.WithResolution(...)", Path},
		{"stocks/candles", "date", "date", "window", "single day", "stocks.WithCandleWindow(stocks.OnDate(d))", q},
		{"stocks/candles", "from", "date", "window", "range start", "stocks.WithCandleWindow(stocks.Between/Since)", q},
		{"stocks/candles", "to", "date", "window", "range end", "stocks.WithCandleWindow(stocks.Between/Until/LastNUntil)", q},
		{"stocks/candles", "countback", "int", "window", "N most recent", "stocks.WithCandleWindow(stocks.LastN)", q},
		{"stocks/candles", "extended", "bool", "", "extended-hours", "stocks.WithCandleExtended(true)", q},
		{"stocks/candles", "adjustsplits", "bool", "", "split adjustment", "stocks.WithCandleAdjustSplits(true)", q},
		{"stocks/candles", "adjustdividends", "bool", "", "dividend adjustment (default true)", "stocks.WithCandleAdjustDividends(false)", q},
		// stocks/bulkcandles
		{"stocks/bulkcandles", "symbols", "[]string", "", "comma-separated tickers; omitted entirely for the market-wide snapshot", "Stocks.BulkCandles(ctx, symbols) / BulkCandles(ctx, nil, stocks.WithSnapshot(true))", q},
		{"stocks/bulkcandles", "resolution", "enum", "", "D only", "stocks.WithBulkResolution(...)", Path},
		{"stocks/bulkcandles", "date", "date", "", "historical day", "stocks.WithBulkDate(d)", q},
		{"stocks/bulkcandles", "adjustsplits", "bool", "", "split adjustment", "stocks.WithAdjustSplits(true)", q},
		{"stocks/bulkcandles", "adjustdividends", "bool", "", "dividend adjustment (default true)", "stocks.WithAdjustDividends(false)", q},
		{"stocks/bulkcandles", "snapshot", "bool", "", "latest-candle snapshot", "stocks.WithSnapshot(true)", q},
		// stocks/prices
		{"stocks/prices", "symbols", "[]string", "", "comma-separated tickers", "Stocks.Prices(ctx, symbols)", q},
		{"stocks/prices", "extended", "bool", "", "extended-hours", "stocks.WithPriceExtended(true)", q},
		// stocks/earnings
		{"stocks/earnings", "symbol", "string", "", "underlying ticker", "Stocks.Earnings(ctx, symbol)", Path},
		{"stocks/earnings", "date", "date", "window", "single day", "stocks.WithEarningsWindow(stocks.OnDate(d))", q},
		{"stocks/earnings", "from", "date", "window", "range start", "stocks.WithEarningsWindow(stocks.Between/Since)", q},
		{"stocks/earnings", "to", "date", "window", "range end", "stocks.WithEarningsWindow(stocks.Between/Until)", q},
		{"stocks/earnings", "countback", "int", "window", "N most recent", "stocks.WithEarningsWindow(stocks.LastN)", q},
		{"stocks/earnings", "report", "string", "", "e.g. 2024-Q1; declared by the schema but inert API-side (live probe 2026-08-11 returned the default row unchanged)", "stocks.WithEarningsReport(...)", q},
		// stocks/news
		{"stocks/news", "symbol", "string", "", "underlying ticker", "Stocks.News(ctx, symbol)", Path},
		{"stocks/news", "date", "date", "window", "single day", "stocks.WithNewsWindow(stocks.OnDate(d))", q},
		{"stocks/news", "from", "date", "window", "range start", "stocks.WithNewsWindow(stocks.Between/Since)", q},
		{"stocks/news", "to", "date", "window", "range end", "stocks.WithNewsWindow(stocks.Between/Until)", q},
		{"stocks/news", "countback", "int", "window", "N most recent", "stocks.WithNewsWindow(stocks.LastN)", q},
	}
}

func fundsParams() []Param {
	const q = Query
	return []Param{
		{"funds/candles", "symbol", "string", "", "fund ticker", "Funds.Candles(ctx, symbol)", Path},
		{"funds/candles", "resolution", "enum", "", "D,W,M,Y (no quarterly)", "funds.WithResolution(...)", Path},
		{"funds/candles", "date", "date", "window", "single day (was unreachable pre-hardening)", "funds.WithCandleWindow(funds.OnDate(d))", q},
		{"funds/candles", "from", "date", "window", "range start", "funds.WithCandleWindow(funds.Between/Since)", q},
		{"funds/candles", "to", "date", "window", "range end", "funds.WithCandleWindow(funds.Between/Until/LastNUntil)", q},
		{"funds/candles", "countback", "int", "window", "N most recent", "funds.WithCandleWindow(funds.LastN)", q},
	}
}

func optionsParams() []Param {
	const q = Query
	return []Param{
		// options/chain
		{"options/chain", "symbol", "string", "", "underlying ticker", "Options.Chain(ctx, symbol)", Path},
		{"options/chain", "expiration", "date|all", "expiry", "exact expiration, or \"all\" for every listed expiration (unfiltered returns only the front month)", "options.WithExpiry(options.OnExpiration(t)/options.AllExpirations())", q},
		{"options/chain", "dte", "int", "expiry", "days to expiry", "options.WithExpiry(options.InDTE(n))", q},
		{"options/chain", "month", "int", "expiry", "1..12", "options.WithExpiry(options.InMonth/InMonthOfYear)", q},
		{"options/chain", "year", "int", "expiry", "4-digit", "options.WithExpiry(options.InYear/InMonthOfYear)", q},
		{"options/chain", "strike", "string", "strike", "exact/list/range/>=,<=", "options.WithStrike(options.Strike/Strikes/StrikeRange/Min/Max/Expr)", q},
		{"options/chain", "delta", "float|list", "strike", "|delta| filter, non-zero in [-1,1]; returns both sides", "options.WithStrike(options.ByDelta(d)/options.ByDeltas(c,d))", q},
		{"options/chain", "date", "date", "", "historical chain as-of a day", "options.WithChainDate(d)", q},
		{"options/chain", "from", "date", "expiry", "expiration range start", "options.WithExpiry(options.ExpirationBetween(a,b))", q},
		{"options/chain", "to", "date", "expiry", "expiration range end", "options.WithExpiry(options.ExpirationBetween(a,b))", q},
		{"options/chain", "side", "enum", "", "call|put", "options.WithSide(...)", q},
		{"options/chain", "strikeLimit", "int", "", "n strikes on EACH side of the money, so up to 2n distinct", "options.WithStrikeLimit(n)", q},
		{"options/chain", "range", "enum", "", "itm|otm|all", "options.WithRange(...)", q},
		{"options/chain", "minBid", "float", "", "min bid", "options.WithMinBid(x)", q},
		{"options/chain", "maxBid", "float", "", "max bid", "options.WithMaxBid(x)", q},
		{"options/chain", "minAsk", "float", "", "min ask", "options.WithMinAsk(x)", q},
		{"options/chain", "maxAsk", "float", "", "max ask", "options.WithMaxAsk(x)", q},
		{"options/chain", "maxBidAskSpread", "float", "", "max abs spread", "options.WithMaxBidAskSpread(x)", q},
		{"options/chain", "maxBidAskSpreadPct", "float", "", "max pct spread", "options.WithMaxBidAskSpreadPct(x)", q},
		{"options/chain", "minOpenInterest", "int", "", "min OI", "options.WithMinOpenInterest(n)", q},
		{"options/chain", "minVolume", "int", "", "min volume", "options.WithMinVolume(n)", q},
		{"options/chain", "weekly", "bool", "exptype", "include/exclude weeklies", "options.WithExpirationTypes(options.IncludeExpirationTypes(options.Weekly))", q},
		{"options/chain", "monthly", "bool", "exptype", "include/exclude monthlies", "options.WithExpirationTypes(options.IncludeExpirationTypes(options.Monthly))", q},
		{"options/chain", "quarterly", "bool", "exptype", "include/exclude quarterlies", "options.WithExpirationTypes(options.IncludeExpirationTypes(options.Quarterly))", q},
		{"options/chain", "nonstandard", "bool", "", "nonstandard contracts", "options.WithNonstandard(true)", q},
		{"options/chain", "am", "bool", "", "AM-settled", "options.WithAM(true)", q},
		{"options/chain", "pm", "bool", "", "PM-settled", "options.WithPM(true)", q},
		// options/quotes (single contract)
		{"options/quotes", "optionSymbol", "string", "", "OCC symbol", "Options.Quote(ctx, optionSymbol)", Path},
		{"options/quotes", "date", "date", "window", "historical day", "options.WithOptionQuoteWindow(options.QuoteOnDate(d))", q},
		{"options/quotes", "from", "date", "window", "range start", "options.WithOptionQuoteWindow(options.QuoteRange)", q},
		{"options/quotes", "to", "date", "window", "range end / countback anchor", "options.WithOptionQuoteWindow(options.QuoteRange/QuoteLastNUntil)", q},
		{"options/quotes", "countback", "int", "window", "N most recent quotes", "options.WithOptionQuoteWindow(options.QuoteLastN/QuoteLastNUntil)", q},
		// options/expirations
		{"options/expirations", "symbol", "string", "", "underlying ticker", "Options.Expirations(ctx, symbol)", Path},
		{"options/expirations", "strike", "float", "", "limit to strike", "options.WithExpirationStrike(x)", q},
		{"options/expirations", "date", "date", "", "expirations as-of day", "options.WithExpirationDate(d)", q},
		// options/lookup
		{"options/lookup", "query", "string", "", "human contract query as path", "Options.Lookup(ctx, underlying, exp, strike, type)", Path},
	}
}

func marketsParams() []Param {
	const q = Query
	return []Param{
		{"markets/status", "date", "date", "", "single day", "markets.WithDate(d)", q},
		{"markets/status", "country", "string", "", "ISO-3166 alpha-2", "markets.WithCountry(c)", q},
		{"markets/status-history", "from", "date", "window", "range start", "markets.WithHistoryWindow(markets.Between/Since)", q},
		{"markets/status-history", "to", "date", "window", "range end", "markets.WithHistoryWindow(markets.Between/Until/LastNUntil)", q},
		{"markets/status-history", "countback", "int", "window", "N most recent days", "markets.WithHistoryWindow(markets.LastN)", q},
		{"markets/status-history", "country", "string", "", "ISO-3166 alpha-2", "markets.WithCountry(c)", q},
	}
}

func utilitiesParams() []Param {
	// The utilities endpoints (status, headers, user) take no request
	// parameters. Listed for completeness of endpoint coverage.
	return []Param{
		{"utilities/status", "-", "-", "", "no parameters", "Utilities.Status(ctx)", Path},
		{"utilities/headers", "-", "-", "", "no parameters", "Utilities.Headers(ctx)", Path},
		{"utilities/user", "-", "-", "", "no parameters", "Utilities.User(ctx)", Path},
	}
}

func universalParams() []Param {
	return []Param{
		{"universal", "columns", "[]string", "", "filter columns (safe with typed decode)", "marketdata.WithColumns(...)", Query},
		{"universal", "mode", "enum", "", "live|cached|delayed (premium; renamed from feed)", "marketdata.WithMode(...)", Query},
		{"universal", "maxage", "string", "", "max cached-data age with mode=cached (e.g. 5min)", "marketdata.WithMaxAge(...)", Query},
		{"universal", "limit", "int", "", "cap results / override endpoint default", "marketdata.WithLimit(...)", Query},
		{"universal", "offset", "int", "", "pagination offset (with limit)", "marketdata.WithOffset(...)", Query},
		{"universal", "dateformat", "enum", "", "SDK-owned for typed decode; advanced override", "marketdata.WithDateFormat(...)", Residual},
		{"universal", "human", "bool", "", "human-readable; incompatible with typed decode", "marketdata.WithHumanReadable(...)", Residual},
		{"universal", "headers", "bool", "", "CSV-only; no effect on JSON responses (has an effect via the CSV facet, ADR-018 — out of this catalog's typed-decode scope)", "marketdata.WithAddHeaders(...)", Residual},
		{"universal", "format", "enum", "", "json|csv|html; SDK requires JSON for typed decoding (csv/html reachable only via each service's AsCSV()/unexported asHTML() facet, ADR-018 — out of this catalog's typed-decode scope)", "SDK-owned (JSON) for typed methods; csv/html set by the CSV/HTML facets, not a user-facing option", Residual},
	}
}
