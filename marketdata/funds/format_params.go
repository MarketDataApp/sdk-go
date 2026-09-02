package funds

import (
	"net/url"

	"github.com/MarketDataApp/sdk-go/v2/internal/http"
	"github.com/MarketDataApp/sdk-go/v2/internal/sdkerrors"
)

// candlesPath is the single serializer for the funds candles request:
// Service.Candles and both formatted facets — CSV (exported) and HTML
// (unexported) — all build their request here, so they can only differ in
// wire format and response type. See ADR-018 for the facets.
func candlesPath(symbol string, opts []CandleOption) (string, url.Values, error) {
	if symbol == "" {
		return "", nil, &sdkerrors.ValidationError{Field: "symbol", Message: "symbol is required"}
	}

	options := defaultCandleOptions()
	for _, opt := range opts {
		opt.apply(options)
	}
	if err := options.window.Validate(); err != nil {
		return "", nil, err
	}

	p := url.Values{}
	options.window.Apply(p)
	return "funds/candles/" + http.PathSegment(options.resolution.String()) + "/" + http.PathSegment(symbol) + "/", p, nil
}
