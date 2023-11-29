package client

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"fmt"
	"strings"
	"time"
)

func extractRateLimitHeaders(resp *http.Response) (limit *int, remaining *int, reset *int64, err error) {
	limitHeader := resp.Header.Get("X-Api-Ratelimit-Limit")
	remainingHeader := resp.Header.Get("X-Api-Ratelimit-Remaining")
	resetHeader := resp.Header.Get("X-Api-Ratelimit-Reset")

	if limitHeader == "" || remainingHeader == "" || resetHeader == "" {
		fmt.Println("URL of the request made: ", resp.Request.URL)
		return nil, nil, nil, errors.New("missing rate limit headers")
	}

	limitVal, err := strconv.Atoi(limitHeader)
	if err != nil {
		return nil, nil, nil, err
	}

	remainingVal, err := strconv.Atoi(remainingHeader)
	if err != nil {
		return nil, nil, nil, err
	}

	resetVal, err := strconv.ParseInt(resetHeader, 10, 64)
	if err != nil {
		return nil, nil, nil, err
	}

	return &limitVal, &remainingVal, &resetVal, nil
}

func GetRateLimitConsumed(resp *http.Response) (int, error) {
	rateLimitConsumed, err := strconv.Atoi(resp.Header.Get("X-Api-RateLimit-Consumed"))
	if err != nil {
		return 0, err
	}
	return rateLimitConsumed, nil
}

func validatePath(path string) (string, error) {
	// Ensure the path starts with a "/"
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Ensure the path ends with a "/"
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// Parse and validate the path
	parsedPath, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return parsedPath.String(), nil
}

func validateQuery(query string) (string, error) {
	// Ensure the query does not start with a "/"
	query = strings.TrimPrefix(query, "/")
	
	// Ensure the query starts with a "?"
	if !strings.HasPrefix(query, "?") && query != "" {
		query = "?" + query
	}

	// Parse and validate the query
	parsedQuery, err := url.Parse(query)
	if err != nil {
		return "", err
	}

	// Parse the query parameters into a map
	values, err := url.ParseQuery(parsedQuery.RawQuery)
	if err != nil {
		return "", err
	}

	// Iterate over the map and delete keys with no value
	for key, value := range values {
		if len(value) == 0 || value[0] == "" {
			delete(values, key)
		}
	}

	// Encode the map back into a query string
	parsedQuery.RawQuery = values.Encode()

	return parsedQuery.String(), nil
}

func DecodeDate(date interface{}) (string, error) {
	switch v := date.(type) {
	case time.Time:
		return v.Format("2006-01-02"), nil
	case string:
		_, err := time.Parse("2006-01-02", v)
		if err != nil {
			return "", err
		}
		return v, nil
	default:
		return "", errors.New("date must be a time.Time object or a YYYY-MM-DD string")
	}
}
