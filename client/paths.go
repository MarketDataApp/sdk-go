package client

var Paths = map[int]map[string]map[string]string{
	1: {
		"markets": {
			"status": "/v1/markets/status/",
		},
	},
	2: {
		"stocks": {
			"tickers": "/v2/stocks/tickers/{datekey}/",
			"candles": "/v2/stocks/candles/{datekey}",
		},
	},
}