package models

import (
	"time"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/govalues/decimal"
)

type Clients map[string]ccxt.IExchange

type Exchanges map[string]Exchange

type Exchange struct {
	Id         string
	Currencies map[string]Currency
	Markets    map[string]Market
}

type Currency struct {
	Id            string
	WithdrawalFee decimal.Decimal
	Network       string
}

type Market struct {
	Id        string
	Base      string
	Quote     string
	Ask       decimal.Decimal
	Bid       decimal.Decimal
	TakerFee  decimal.Decimal
	MakerFee  decimal.Decimal
	Timestamp time.Time
}
