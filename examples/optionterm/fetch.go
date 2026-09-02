// The fetch layer: one function per SDK call the program makes. This file
// is the reference customers open to copy a working pattern for any
// options-endpoint call — every function here follows the same shape and
// can be lifted verbatim into another program with the symbol and options
// swapped.
//
// Every fetch function:
//
//  1. Builds a context bounded by fetchTimeout, so no call can hang the
//     program indefinitely.
//  2. Calls exactly one SDK method — nothing else touches the client.
//  3. On error, returns errMsg{op, err}, where op names the operation.
//  4. On success, returns a typed message carrying the SDK result plus the
//     *marketdata.Response metadata (rate-limit info, raw headers, etc.).
//
// A 404 (no data available) is not an error: the SDK methods return a nil
// (or empty) result, a nil error, and a *marketdata.Response whose NoData
// field is true. That response still flows through the success path as a
// typed message, not errMsg — callers branch on NoData or an empty result,
// never on err, to tell "nothing found" apart from "the request failed."
package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MarketDataApp/sdk-go/v2/marketdata"
	"github.com/MarketDataApp/sdk-go/v2/marketdata/options"
)

// fetchTimeout bounds every SDK call made by the app.
const fetchTimeout = 15 * time.Second

// fetchUnderlying loads the current quote for the active underlying symbol
// via [stocks.Service.Quote] (client.Stocks.Quote), the stocks/bulkquotes
// endpoint. It takes no options.
func fetchUnderlying(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		quote, resp, err := client.Stocks.Quote(ctx, symbol)
		if err != nil {
			return errMsg{op: "underlying", err: err}
		}
		return underlyingMsg{symbol: symbol, quote: quote, meta: resp}
	}
}

// fetchExpirations loads the expiration dates with listed contracts for
// symbol via [options.Service.Expirations] (client.Options.Expirations),
// the options/expirations endpoint. It passes no [options.ExpirationOption]
// values; the SDK always sends dateformat=unix on this endpoint regardless.
func fetchExpirations(client *marketdata.Client, symbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		exps, resp, err := client.Options.Expirations(ctx, symbol)
		if err != nil {
			return errMsg{op: "expirations", err: err}
		}
		var dates []time.Time
		if exps != nil {
			dates = exps.Dates
		}
		return expirationsMsg{symbol: symbol, expirations: dates, meta: resp}
	}
}

// fetchChain loads the option chain for symbol's exp expiration via
// [options.Service.Chain] (client.Options.Chain), the options/chain
// endpoint. It always passes options.WithExpiry(options.OnExpiration); it
// adds options.WithStrike(options.StrikeRange) when both lo and hi are
// greater than zero, and [options.WithSide] when side is not
// [options.SideBoth].
func fetchChain(client *marketdata.Client, symbol string, exp time.Time, lo, hi float64, side options.OptionSide) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		opts := []options.ChainOption{options.WithExpiry(options.OnExpiration(exp))}
		if lo > 0 && hi > 0 {
			opts = append(opts, options.WithStrike(options.StrikeRange(lo, hi)))
		}
		if side != options.SideBoth {
			opts = append(opts, options.WithSide(side))
		}
		chain, resp, err := client.Options.Chain(ctx, symbol, opts...)
		if err != nil {
			return errMsg{op: "chain", err: err}
		}
		return chainMsg{chain: chain, meta: resp}
	}
}

// fetchContract loads a single contract's quote for occSymbol via
// [options.Service.Quote] (client.Options.Quote), the options/quotes
// endpoint. It takes no options.
func fetchContract(client *marketdata.Client, occSymbol string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		quote, resp, err := client.Options.Quote(ctx, occSymbol)
		if err != nil {
			return errMsg{op: "contract", err: err}
		}
		return contractMsg{quote: quote, meta: resp}
	}
}

// fetchPinned loads quotes for every pinned OCC symbol in occs via
// [options.Service.Quotes] (client.Options.Quotes), the options/quotes
// endpoint. Quotes fans out one concurrent request per symbol through the
// client's shared concurrency pool (at most 50 in-flight requests per
// client across every service), so pinning many contracts never issues
// more requests at once than that pool allows, regardless of what else the
// client is doing concurrently. Symbols with no data are silently omitted
// from the result rather than producing an error.
func fetchPinned(client *marketdata.Client, occs []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		quotes, resp, err := client.Options.Quotes(ctx, occs)
		if err != nil {
			return errMsg{op: "pinned", err: err}
		}
		return pinnedMsg{quotes: quotes, meta: resp}
	}
}

// lookupContract resolves an OCC option symbol for underlying, exp, strike,
// and typ via [options.Service.Lookup] (client.Options.Lookup), the
// options/lookup endpoint. Unlike every other options endpoint, Lookup
// encodes the whole human-readable query (underlying, date, strike, and
// type) as a single URL path segment rather than as query parameters.
func lookupContract(client *marketdata.Client, underlying string, exp time.Time, strike float64, typ options.OptionType) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()
		occ, resp, err := client.Options.Lookup(ctx, underlying, exp, strike, typ)
		if err != nil {
			return errMsg{op: "lookup", err: err}
		}
		return lookupMsg{occ: occ, noData: resp != nil && resp.NoData, meta: resp}
	}
}
