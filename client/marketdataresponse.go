package client

import (
	"errors"
	"log"
	"strconv"

	"github.com/go-resty/resty/v2"
)

// MarketDataResponse represents the response from the Market Data API.
// It embeds the resty.Response and includes additional fields for RayID and RateLimitConsumed.
type MarketDataResponse struct {
	*resty.Response          // The response from the resty client
	RayID             string // The Ray ID from the HTTP response
	RateLimitConsumed int    // The number of requests consumed from the rate limit
	Delay             int64  // The time (in miliseconds) between the request and the server's response
}

func (mdr *MarketDataResponse) setLatency() {
	trace := mdr.Request.TraceInfo()
	mdr.Delay = trace.ServerTime.Milliseconds()
}

// GetRateLimitConsumed retrieves the number of requests consumed from the HTTP response.
// It sets the number of requests consumed to the struct and returns an error if any.
func (mdr *MarketDataResponse) setRateLimitConsumed() {
	rateLimitConsumedStr := mdr.Response.Header().Get("X-Api-RateLimit-Consumed")
	if rateLimitConsumedStr == "" {
		log.Println("Error: missing 'x-Api-RateLimit-Consumed' header")
		return
	}
	rateLimitConsumed, err := strconv.Atoi(rateLimitConsumedStr)
	if err != nil {
		log.Println(err)
		return
	}
	mdr.RateLimitConsumed = rateLimitConsumed
}

// GetRayID retrieves the Cf-Ray ID from the HTTP response.
// It sets the Cf-Ray ID to the struct and returns an error if any.
func (mdr *MarketDataResponse) setRayID() error {
	rayID := mdr.Response.Header().Get("Cf-Ray")
	if rayID == "" {
		return errors.New("Cf-Ray header not found")
	}
	mdr.RayID = rayID
	return nil
}

func (c *MarketDataClient) Get(path string) (*resty.Response, error) {
	req := c.Client.R()
	return c.wrapResponse(req, path)
}

func (c *MarketDataClient) wrapResponse(req *resty.Request, path string) (*resty.Response, error) {
	resp, err := req.Get(path) // Must run GET after setting all params.
	if err != nil {
		return nil, err
	}
	/*
	mdr := &MarketDataResponse{Response: resp}
	mdr.setRateLimitConsumed()

	err = mdr.setRayID()
	if err != nil {
		return nil, err
	}

	mdr.setLatency()
	*/
	return resp, nil
}
