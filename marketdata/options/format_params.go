package options

import (
	"net/url"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// Request path and query parameter construction for every options
// endpoint. These are the single serializer: the JSON methods on [Service]
// and both formatted facets — CSV (exported, [Service.AsCSV]) and HTML
// (unexported, [Service.asHTML]) — all build their request here, so the
// three can only ever differ in wire format and response type, never in
// what they ask the API for. See ADR-018 for the facets.
//
// The one deliberate exception is dateformat on expirations, which serves a
// decoder rather than the request — see expirationsPath.
//
// The config locals are named cfg, not options: inside package options a
// local called options reads as a package-qualified selector.

func chainPath(symbol string, opts []ChainOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	cfg := defaultChainOptions()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	p := url.Values{}
	if cfg.strike != nil {
		sp := cfg.strike.strike()
		if err := sp.validate(); err != nil {
			return "", nil, err
		}
		sp.apply(p)
	}
	if cfg.expiry != nil {
		ep := cfg.expiry.expiry()
		if err := ep.validate(); err != nil {
			return "", nil, err
		}
		ep.apply(p)
	}
	if !cfg.date.IsZero() {
		p.Set("date", cfg.date.Format("2006-01-02"))
	}
	if cfg.side != "" {
		p.Set("side", string(cfg.side))
	}
	if cfg.expTypes != nil {
		include, types := cfg.expTypes.expirationTypes()
		for _, t := range types {
			p.Set(string(t), formatBool(include))
		}
	}
	if cfg.strikeLimit > 0 {
		p.Set("strikeLimit", formatInt(cfg.strikeLimit))
	}
	if cfg.rangeFilter != "" {
		p.Set("range", string(cfg.rangeFilter))
	}
	if cfg.minBid != nil {
		p.Set("minBid", formatFloat(*cfg.minBid))
	}
	if cfg.maxBid != nil {
		p.Set("maxBid", formatFloat(*cfg.maxBid))
	}
	if cfg.minAsk != nil {
		p.Set("minAsk", formatFloat(*cfg.minAsk))
	}
	if cfg.maxAsk != nil {
		p.Set("maxAsk", formatFloat(*cfg.maxAsk))
	}
	if cfg.maxBidAskSpread != nil {
		p.Set("maxBidAskSpread", formatFloat(*cfg.maxBidAskSpread))
	}
	if cfg.maxBidAskSpreadPct != nil {
		p.Set("maxBidAskSpreadPct", formatFloat(*cfg.maxBidAskSpreadPct))
	}
	if cfg.minOpenInterest != nil {
		p.Set("minOpenInterest", formatInt(*cfg.minOpenInterest))
	}
	if cfg.minVolume != nil {
		p.Set("minVolume", formatInt(*cfg.minVolume))
	}
	if cfg.nonstandard != nil {
		p.Set("nonstandard", formatBool(*cfg.nonstandard))
	}
	if cfg.am != nil {
		p.Set("am", formatBool(*cfg.am))
	}
	if cfg.pm != nil {
		p.Set("pm", formatBool(*cfg.pm))
	}

	return "options/chain/" + http.PathSegment(symbol) + "/", p, nil
}

func expirationsPath(symbol string, opts []ExpirationOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	cfg := &expirationOptions{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	// No dateformat here, unlike Service.Expirations: that method forces
	// dateformat=unix to feed its JSON decoder, but this facet returns the
	// API's text verbatim, so the wire format IS the user-visible output.
	// Setting it would also defeat WithDateFormat/MARKETDATA_DATE_FORMAT,
	// since Do() merges client defaults only for absent keys. The "mirrors
	// the JSON method exactly" rule above governs filter parameters; a
	// decoder-serving parameter belongs only on the decoding path.
	p := url.Values{}
	if cfg.strike > 0 {
		p.Set("strike", formatFloat(cfg.strike))
	}
	if !cfg.date.IsZero() {
		p.Set("date", cfg.date.Format("2006-01-02"))
	}

	return "options/expirations/" + http.PathSegment(symbol) + "/", p, nil
}

func quotePath(optionSymbol string, opts []QuoteOption) (string, url.Values, error) {
	if optionSymbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "optionSymbol", Message: "option symbol is required"}
	}

	cfg := &quoteOptions{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	p := url.Values{}
	if cfg.window != nil {
		w := cfg.window.optionQuoteWindow()
		if err := w.validate("date"); err != nil {
			return "", nil, err
		}
		w.apply(p)
	}

	return "options/quotes/" + http.PathSegment(optionSymbol) + "/", p, nil
}
