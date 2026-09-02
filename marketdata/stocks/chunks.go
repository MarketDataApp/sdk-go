package stocks

import (
	"time"

	"github.com/MarketDataApp/sdk-go/v2/internal/params"
)

// dateChunk is one disjoint slice of a candleChunks split.
type dateChunk struct {
	from, to time.Time
}

// candleChunks splits a bounded date range into disjoint year-sized chunks.
// The API treats date-only from/to as inclusive on both ends, so consecutive
// chunks must not share a boundary day: each chunk covers one inclusive
// calendar year and the next one starts the following day. Sharing the
// boundary (as sibling SDKs do) fetches the boundary day twice and silently
// duplicates its data in a merged result.
//
// Shared between the JSON path ([Service.candlesSplit]) and the CSV/HTML
// facets' Candles, which merge text instead of typed values but need the
// exact same disjoint boundaries.
func candleChunks(window params.Window) []dateChunk {
	var chunks []dateChunk
	from, to := window.From(), window.To()
	// Compare and advance by calendar date only (each value's own Y/M/D, in
	// its own location) — exactly what window.Apply serializes to the wire
	// as YYYY-MM-DD. Comparing full instants instead silently dropped the
	// final day whenever from's wall-clock time or zone offset put it later
	// than to's (e.g. an intraday from at 09:30 against a midnight to, or a
	// from/to pair in different zones where "midnight" isn't simultaneous):
	// current.After(to) went true one iteration early and the last calendar
	// day never got a chunk.
	current := from
	for !afterDate(current, to) {
		end := current.AddDate(1, 0, -1)
		if afterDate(end, to) {
			end = to
		}
		chunks = append(chunks, dateChunk{from: current, to: end})
		current = end.AddDate(0, 0, 1)
	}
	return chunks
}
