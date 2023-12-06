package client

import "github.com/go-resty/resty/v2"

type MarketDataRequest interface {
	GetParams() []MarketDataParam
	GetPath() string
	GetError() error
	GetResty() *resty.Request

}

type MarketDataParam interface {
	SetParams(*resty.Request) error
}
