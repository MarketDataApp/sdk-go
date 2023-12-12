package client

import "github.com/go-resty/resty/v2"

type MarketDataPacked interface {
	IsValid() bool
}

type MarketDataParam interface {
	SetParams(*resty.Request) error
}
