package client

import "github.com/go-resty/resty/v2"

type MarketDataPacked interface {
	IsValid() bool
	Unpack() any
}

type MarketDataParam interface {
	SetParams(*resty.Request) error
}
