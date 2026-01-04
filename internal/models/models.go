package models

import (
	"time"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/govalues/decimal"
)

type CurrencyInputMode int

const (
	AllCurrencies CurrencyInputMode = iota
	SpecifiedCurrencies
	RandomCurrencies
)

type Config struct {
	Exchanges          []string
	CurrencyInputMode  CurrencyInputMode
	Currencies         []string
	ExcludedCurrencies []string
	ReferenceAsset     AssetBalance
	SourceAssets       map[AssetKey]AssetBalance
}

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
	Timestamp time.Time
}

type AssetIndex struct {
	Asset AssetKey
	Index uint
}

type AssetBalance struct {
	Asset   AssetKey
	Balance decimal.Decimal
}

type AssetKey struct {
	Exchange string
	Currency string
}

type AssetIndexes map[AssetKey]AssetIndex

type Index map[uint]AssetKey

type Pair struct {
	IntraExchange bool
	Symbol        string
	From          AssetIndex
	To            AssetIndex
	Weight        decimal.Decimal
	Side          string // can be empty string, if inter-exchange
	Network       string // can be empty string, if intra-exchange
}

type PairKey struct {
	From AssetKey
	To   AssetKey
}

type Pairs map[PairKey]Pair

type TransactionPath []PairKey

// ArbitragePath is a complete representation of an arbitrage path
// ToCycle describes the cheapest/shortest path from the optimal asset in SourceAssets to the optimal cycle asset
// Cycle describes the arbitrage cycle loop of assets
// FromCycle describes the cheapest/shortest path from the optimal cycle asset to the optimal asset in SourceAssets
type ArbitragePath struct {
	ToCycle   TransactionPath
	Cycle     TransactionPath
	FromCycle TransactionPath
}
