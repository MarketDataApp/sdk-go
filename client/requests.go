package client

type MarketDataRequest interface {
	GetPath() (string, error)
	GetQuery() (string, error)
}
