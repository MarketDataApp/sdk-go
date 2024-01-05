package client

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
)

func getRateLimitConsumed(resp *resty.Response) (int, error) {
	rateLimitConsumedStr := resp.Header().Get("X-Api-RateLimit-Consumed")
	if rateLimitConsumedStr == "" {
		return 0, errors.New("error: missing 'x-Api-RateLimit-Consumed' header")
	}
	rateLimitConsumed, err := strconv.Atoi(rateLimitConsumedStr)
	if err != nil {
		return 0, err
	}
	return rateLimitConsumed, nil
}

func getRayIDFromResponse(resp *resty.Response) (string, error) {
	rayID := resp.Header().Get("Cf-Ray")
	if rayID == "" {
		return "", errors.New("Cf-Ray header not found")
	}
	return rayID, nil
}

func getLatencyFromRequest(req *resty.Request) int64 {
	trace := req.TraceInfo()
	return trace.ServerTime.Milliseconds()
}

func redactAuthorizationHeader(headers http.Header) http.Header {
	// Copy the headers so we don't modify the original
	copiedHeaders := make(http.Header)
	for k, v := range headers {
		copiedHeaders[k] = v
	}

	// Redact the Authorization header if it exists
	if _, ok := copiedHeaders["Authorization"]; ok {
		token := copiedHeaders.Get("Authorization")
		redactedToken := "Bearer " + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
		copiedHeaders.Set("Authorization", redactedToken)
	}

	return copiedHeaders
}
