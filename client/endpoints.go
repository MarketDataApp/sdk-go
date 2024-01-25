package client

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
