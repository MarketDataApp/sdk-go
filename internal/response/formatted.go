package response

import (
	"bytes"
	"context"
	"net/url"

	"github.com/MarketDataApp/sdk-go/v2/internal/fanout"
	internalhttp "github.com/MarketDataApp/sdk-go/v2/internal/http"
)

// FetchFormatted issues a GET requesting a non-JSON wire format (format=csv
// or format=html, see ADR-018) and wraps the raw response with wrap (NewCSV
// or NewHTML). Shared by every service's CSV (exported) and HTML
// (unexported) facets so the fetch-and-wrap boilerplate exists exactly
// once.
func FetchFormatted[R any](ctx context.Context, httpClient *internalhttp.Client, path string, params url.Values, format string, wrap func(*internalhttp.Response) R) (R, error) {
	httpResp, err := httpClient.GetFormatted(ctx, path, params, format)
	if err != nil {
		var zero R
		return zero, err
	}
	return wrap(httpResp), nil
}

// FetchFormattedChunked issues one request per element of chunkParams
// concurrently against the same path, cancelling the remaining requests on
// the first error (ADR-014, mirrored from the JSON candle-splitting path),
// then merges the raw bodies in order and wraps the merged text with the
// response metadata of the last chunk that carried data — the same contract
// the JSON path documents. A single chunk skips the concurrency machinery
// entirely and behaves exactly like FetchFormatted.
//
// Chunks the API answered with no data are dropped before the merge, the
// way the JSON path filters NoData chunks. Without that they corrupt the
// output twice over: their placeholder lines land in the merged text, and
// because mergeTextChunks takes its reference header from the first body,
// an empty oldest chunk (the likeliest one, since it predates the listing)
// makes every later chunk keep its own header row inline. See noDataChunk
// for why the detection has to be structural.
func FetchFormattedChunked[R any](ctx context.Context, httpClient *internalhttp.Client, path string, chunkParams []url.Values, format string, wrap func(*internalhttp.Response) R) (R, error) {
	var zero R
	if len(chunkParams) == 1 {
		return FetchFormatted(ctx, httpClient, path, chunkParams[0], format, wrap)
	}

	resps, err := fanout.Run(ctx, len(chunkParams), func(ctx context.Context, i int) (*internalhttp.Response, error) {
		return httpClient.GetFormatted(ctx, path, chunkParams[i], format)
	})
	if err != nil {
		return zero, err
	}

	bodies := make([][]byte, 0, len(resps))
	var lastWithData *internalhttp.Response
	for _, r := range resps {
		if noDataChunk(r.Body) {
			continue
		}
		bodies = append(bodies, r.Body)
		lastWithData = r
	}
	if lastWithData == nil {
		// Every chunk was empty. There is no NoData concept in these
		// formats (ADR-018), so hand back the API's own no-data body rather
		// than inventing an empty one: the caller sees what it sent.
		return wrap(resps[len(resps)-1]), nil
	}

	lastWithData.Body = mergeTextChunks(bodies)
	return wrap(lastWithData), nil
}

// FetchFormattedMap issues one request per key concurrently against a
// per-key path/params (built by requestFor, which may reject a key whose
// options do not validate), cancelling the rest on the first error
// (ADR-014) — used by facets whose JSON counterpart fans out
// per-symbol into a slice instead of merging into one result (e.g.
// options.Quotes). Unlike the JSON path, there is no NoData omission: every
// key gets an entry in the returned map holding whatever the API sent for
// it, since these facets have no NoData concept (see ADR-018).
func FetchFormattedMap[K comparable, R any](ctx context.Context, httpClient *internalhttp.Client, keys []K, requestFor func(K) (path string, params url.Values, err error), format string, wrap func(*internalhttp.Response) R) (map[K]R, error) {
	if len(keys) == 1 {
		path, params, err := requestFor(keys[0])
		if err != nil {
			return nil, err
		}
		r, err := FetchFormatted(ctx, httpClient, path, params, format, wrap)
		if err != nil {
			return nil, err
		}
		return map[K]R{keys[0]: r}, nil
	}

	resps, err := fanout.Run(ctx, len(keys), func(ctx context.Context, i int) (R, error) {
		path, params, err := requestFor(keys[i])
		if err != nil {
			var zero R
			return zero, err
		}
		return FetchFormatted(ctx, httpClient, path, params, format, wrap)
	})
	if err != nil {
		return nil, err
	}

	out := make(map[K]R, len(keys))
	for i, key := range keys {
		out[key] = resps[i]
	}
	return out, nil
}

// NoDataBody and NoDataBodyNoHeaders are the exact bodies production
// returns for a formatted (CSV) request that finds no data — with headers
// on (the default) and with headers=false — captured 2026-08-19 against
// /v1/stocks/candles/D/RIVN/ for 2015 and re-verified 2026-08-25. The CRLF
// line endings are the API's own. They are the single source for the shape
// noDataChunk recognizes: the unit fixtures consume them verbatim, and
// integration's TestWireShape_FormattedNoData re-fetches them live — so if
// the API changes the shape, a test fails instead of the fail-open filter
// silently ceasing to match.
const (
	NoDataBody          = "0\r\n\"\"\r\n"
	NoDataBodyNoHeaders = "\"\"\r\n"
)

// noDataChunk reports whether a chunk body is the API's "no data" answer
// rather than data. It has to be detected structurally: unlike the JSON
// path, which gets a 404 or an explicit no-data body, the formatted no-data
// response is HTTP 200 carrying no marker at all to search for.
//
// Two shapes, both verified against production on 2026-08-19 with a
// pre-listing candles window:
//
//	0                 <- placeholder where the header row would be
//	""                   (default, headers on)
//
//	""                <- headers=false
//
// So the invariant is the `""` line; the bare `0` is tolerated alongside it
// as the header placeholder. Requiring `""` to be present keeps the match
// tight: a real row of any of these endpoints has commas and values, and
// none can render a whole line as `""`. If the API ever changes the shape
// this stops matching and the behavior falls back to today's — the
// placeholder lines reappear in the output — rather than discarding data.
func noDataChunk(body []byte) bool {
	sawEmptyMarker := false
	for _, line := range bytes.Split(body, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		switch {
		case len(line) == 0:
		case bytes.Equal(line, []byte(`""`)):
			sawEmptyMarker = true
		case bytes.Equal(line, []byte("0")):
		default:
			return false
		}
	}
	return sawEmptyMarker
}

// mergeTextChunks concatenates chunk bodies in request order. When a
// chunk's leading line (up to the first '\n') is identical to the first
// chunk's leading line, it's dropped as a repeated header row — detected
// structurally by comparing lines, not by threading through whether the
// "headers" universal param was set, so it's correct regardless of that
// configuration.
func mergeTextChunks(bodies [][]byte) []byte {
	if len(bodies) == 0 {
		return nil
	}

	header := firstLine(bodies[0])
	var out bytes.Buffer
	for i, b := range bodies {
		body := b
		if i > 0 && bytes.Equal(firstLine(b), header) {
			if nl := bytes.IndexByte(b, '\n'); nl >= 0 {
				body = b[nl+1:]
			} else {
				body = nil
			}
		}
		if len(body) == 0 {
			continue
		}
		if out.Len() > 0 && out.Bytes()[out.Len()-1] != '\n' {
			out.WriteByte('\n')
		}
		out.Write(body)
	}
	return out.Bytes()
}

func firstLine(b []byte) []byte {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		return b[:i]
	}
	return b
}
