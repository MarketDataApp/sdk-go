package client

// endpoints maps API calls to their corresponding endpoints.
var endpoints = map[int]map[string]map[string]string{
	1: {
		"markets": {
			"status": "/v1/markets/status/",
		},
		"stocks": {
			"candles":  "/v1/stocks/candles/{resolution}/{symbol}/",
			"quotes":   "/v1/stocks/quotes/{symbol}/",
			"earnings": "/v1/stocks/earnings/{symbol}/",
			"news":     "/v1/stocks/news/{symbol}/",
		},
		"options": {
			"expirations": "/v1/options/expirations/{symbol}/",
			"lookup":      "/v1/options/lookup/{userInput}",
			"strikes":     "/v1/options/strikes/{symbol}/",
			"quotes":      "/v1/options/quotes/{symbol}/",
			"chain":       "/v1/options/chain/{symbol}/",
		},
		"indices": {
			"quotes":  "/v1/indices/quotes/{symbol}/",
			"candles": "/v1/indices/candles/{resolution}/{symbol}/",
		},
	},
	2: {
		"stocks": {
			"tickers": "/v2/stocks/tickers/{datekey}/",
			"candles": "/v2/stocks/candles/{resolution}/{symbol}/{datekey}",
		},
	},
}
