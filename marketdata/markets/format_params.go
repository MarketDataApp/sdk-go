package markets

import "net/url"

// Request path and query parameter construction for both markets endpoints.
// These are the single serializer: Service.Status, Service.StatusHistory and
// the formatted facets — CSV (exported) and HTML (unexported), which cover
// Status only per ADR-018 — all build their request here.
//
// The two methods share one wire path and one country parameter while
// differing in their date window, which is exactly the shape that produced
// this SDK's drift bugs, where one serializer sends a parameter its twin
// silently drops. StatusHistory used to build its params
// inline, leaving `if country != ""` written twice; nothing had drifted, but
// the structure that lets it drift had no reason to survive.

const statusEndpoint = "markets/status/"

func statusPath(opts []StatusOption) (string, url.Values) {
	options := &statusOptions{}
	for _, opt := range opts {
		opt.applyStatus(options)
	}

	p := url.Values{}
	if !options.date.IsZero() {
		p.Set("date", options.date.Format("2006-01-02"))
	}
	applyCountry(p, options.country)
	return statusEndpoint, p
}

// statusHistoryPath mirrors statusPath for the history method. It has no
// formatted facet (ADR-018 gives StatusHistory no CSV counterpart), so it is
// the one builder here with a single consumer — kept a builder anyway so a
// facet added later cannot be born drifted, the same reasoning applied to
// stocks.bulkCandlesPath.
func statusHistoryPath(opts []HistoryOption) (string, url.Values, error) {
	options := &historyOptions{}
	for _, opt := range opts {
		opt.applyHistory(options)
	}
	if err := options.window.Validate(); err != nil {
		return "", nil, err
	}

	p := url.Values{}
	options.window.Apply(p)
	applyCountry(p, options.country)
	return statusEndpoint, p, nil
}

// applyCountry is the one place the country parameter is serialized, so the
// two builders cannot disagree about it.
func applyCountry(p url.Values, country string) {
	if country != "" {
		p.Set("country", country)
	}
}
