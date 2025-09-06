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

type CurrencyNetwork struct {
	Id            string
	WithdrawalFee decimal.Decimal
}

type Currency struct {
	Id       string
	Networks map[string]CurrencyNetwork
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

type Asset struct {
	Exchange string
	Currency string
	Index    uint
}

type AssetKey struct {
	Exchange string
	Currency string
}

type Assets map[AssetKey]Asset

type Index map[uint]AssetKey

type Pair struct {
	IntraExchange bool
	Symbol        string
	From          Asset
	To            Asset
	Weight        decimal.Decimal
	Side          string // can be empty string, if inter-exchange
	Network       string // can be empty string, if intra-exchange
}

type PairKey struct {
	From AssetKey
	To   AssetKey
}

type Pairs map[PairKey]Pair
