package http

import (
	"net/url"
	"strings"
)

// PathSegment percent-encodes a caller-supplied value for safe use as a
// single URL path segment. It prevents path injection: a value such as
// "AAPL/../user" cannot escape its segment or re-route the request.
// Dot-segments ("." and ".."), which percent-encoding leaves intact because
// dots are unreserved, are neutralized explicitly.
func PathSegment(s string) string {
	escaped := url.PathEscape(s)
	if escaped == "." || escaped == ".." {
		escaped = strings.ReplaceAll(escaped, ".", "%2E")
	}
	return escaped
}
