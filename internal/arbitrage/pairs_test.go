package arbitrage

import (
	"maps"
	"testing"
	"time"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestCreateAssetIndex(t *testing.T) {
	testCases := []struct {
		name          string
		testExchanges models.Exchanges
	}{
		{
			name: "successful creation with multiple exchanges and currencies",
			testExchanges: models.Exchanges{
				"binance": {
					Id: "binance",
					Currencies: map[string]models.Currency{
						"BTC": {},
						"ETH": {},
						"LTC": {},
					},
				},
				"kucoin": {
					Id: "kucoin",
					Currencies: map[string]models.Currency{
						"SOL":  {},
						"XRP":  {},
						"DOGE": {},
					},
				},
			},
		},
		{
			name:          "empty exchanges",
			testExchanges: models.Exchanges{},
		},
		{
			name: "single exchange with one currency",
			testExchanges: models.Exchanges{
				"binance": {
					Id: "binance",
					Currencies: map[string]models.Currency{
						"BTC": {},
					},
				},
			},
		},
		{
			name: "exchanges with overlapping currencies",
			testExchanges: models.Exchanges{
				"binance": {
					Id: "binance",
					Currencies: map[string]models.Currency{
						"BTC": {},
						"ETH": {},
					},
				},
				"kucoin": {
					Id: "kucoin",
					Currencies: map[string]models.Currency{
						"ETH": {},
						"XRP": {},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotAssets, gotIndex := createAssetIndex(&tc.testExchanges)

			var totalCurrencies int
			for _, exchange := range tc.testExchanges {
				totalCurrencies += len(exchange.Currencies)
			}

			if len(gotAssets) != totalCurrencies {
				t.Fatalf("createAssetIndex() returned asset map of size %d, want %d", len(gotAssets), totalCurrencies)
			}

			if len(gotIndex) != totalCurrencies {
				t.Fatalf("createAssetIndex() returned index map of size %d, want %d", len(gotIndex), totalCurrencies)
			}

			for assetKey, asset := range gotAssets {
				if _, ok := gotIndex[asset.Index]; !ok {
					t.Errorf("createAssetIndex() asset map has key %v with index %d, but index is missing in the index map", assetKey, asset.Index)
				}
				if gotIndex[asset.Index] != assetKey {
					t.Errorf("createAssetIndex() index map value at key %d is incorrect.\nGot: %+v\nWant: %+v", asset.Index, gotIndex[asset.Index], assetKey)
				}
			}

			for indexKey, indexValue := range gotIndex {
				if _, ok := gotAssets[indexValue]; !ok {
					t.Errorf("createAssetIndex() index map has key %d with value %+v, but this asset key is missing from the asset map", indexKey, indexValue)
				}
				if gotAssets[indexValue].Index != indexKey {
					t.Errorf("createAssetIndex() asset map for key %+v has incorrect index.\nGot: %d\nWant: %d", indexValue, gotAssets[indexValue].Index, indexKey)
				}
			}
		})
	}
}

func TestCreateIntraExchangePairs(t *testing.T) {
	testCases := []struct {
		name          string
		exchangesPtr  *models.Exchanges
		assetsPtr     *models.Assets
		expectedPairs models.Pairs
	}{
		{
			name: "single exchange with multiple markets",
			exchangesPtr: &models.Exchanges{
				"binance": {
					Id: "binance",
					Markets: map[string]models.Market{
						"BTC/USDC": {
							Id:        "BTC/USDC",
							Base:      "BTC",
							Quote:     "USDC",
							Ask:       decimal.MustNew(11064601, 2), // 110646.01
							Bid:       decimal.MustNew(11064600, 2), // 110646.00
							TakerFee:  decimal.MustNew(1, 3),        // 0.001
							Timestamp: time.Now(),
						},
						"ETH/USDC": {
							Id:        "ETH/USDC",
							Base:      "ETH",
							Quote:     "USDC",
							Ask:       decimal.MustNew(429301, 2), // 4293.01
							Bid:       decimal.MustNew(429300, 2), // 4293.00
							TakerFee:  decimal.MustNew(1, 3),      // 0.001
							Timestamp: time.Now(),
						},
					},
				},
			},
			assetsPtr: &models.Assets{
				models.AssetKey{Exchange: "binance", Currency: "BTC"}:  models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
				models.AssetKey{Exchange: "binance", Currency: "USDC"}: models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
				models.AssetKey{Exchange: "binance", Currency: "ETH"}:  models.Asset{Exchange: "binance", Currency: "ETH", Index: 2},
			},
			expectedPairs: models.Pairs{
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "BTC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "BTC/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
					To:            models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					Weight:        decimal.MustNew(11053535400, 5), // 110646.00 * (1 - 0.001)
					Side:          "sell",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "USDC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "BTC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "BTC/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					To:            models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
					Weight:        decimal.MustNew(90287937179117, 19), // (1/110646.01) * (1 - 0.001)
					Side:          "buy",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "ETH"},
					To:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "ETH/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "ETH", Index: 2},
					To:            models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					Weight:        decimal.MustNew(428870700, 5), // 4293.00 * (1 - 0.001)
					Side:          "sell",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "USDC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "ETH"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "ETH/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					To:            models.Asset{Exchange: "binance", Currency: "ETH", Index: 2},
					Weight:        decimal.MustNew(2327038604615410, 19), // (1/4293.01) * (1 - 0.001)
					Side:          "buy",
				},
			},
		},
		{
			name: "multiple exchanges with markets",
			exchangesPtr: &models.Exchanges{
				"binance": {
					Id: "binance",
					Markets: map[string]models.Market{
						"BTC/USDC": {
							Id:        "BTC/USDC",
							Base:      "BTC",
							Quote:     "USDC",
							Ask:       decimal.MustNew(11064601, 2),
							Bid:       decimal.MustNew(11064600, 2),
							TakerFee:  decimal.MustNew(1, 3),
							Timestamp: time.Now(),
						},
					},
				},
				"kucoin": {
					Id: "kucoin",
					Markets: map[string]models.Market{
						"ETH/USDC": {
							Id:        "ETH/USDC",
							Base:      "ETH",
							Quote:     "USDC",
							Ask:       decimal.MustNew(429334, 2),
							Bid:       decimal.MustNew(429333, 2),
							TakerFee:  decimal.MustNew(1, 3),
							Timestamp: time.Now(),
						},
					},
				},
			},
			assetsPtr: &models.Assets{
				models.AssetKey{Exchange: "binance", Currency: "BTC"}:  models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
				models.AssetKey{Exchange: "binance", Currency: "USDC"}: models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
				models.AssetKey{Exchange: "kucoin", Currency: "ETH"}:   models.Asset{Exchange: "kucoin", Currency: "ETH", Index: 2},
				models.AssetKey{Exchange: "kucoin", Currency: "USDC"}:  models.Asset{Exchange: "kucoin", Currency: "USDC", Index: 3},
			},
			expectedPairs: models.Pairs{
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "BTC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "BTC/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
					To:            models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					Weight:        decimal.MustNew(11053535400, 5),
					Side:          "sell",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "USDC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "BTC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "BTC/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					To:            models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
					Weight:        decimal.MustNew(90287937179117, 19),
					Side:          "buy",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "kucoin", Currency: "ETH"},
					To:   models.AssetKey{Exchange: "kucoin", Currency: "USDC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "ETH/USDC",
					From:          models.Asset{Exchange: "kucoin", Currency: "ETH", Index: 2},
					To:            models.Asset{Exchange: "kucoin", Currency: "USDC", Index: 3},
					Weight:        decimal.MustNew(428903667, 5), // 4293.33 * (1 - 0.001)
					Side:          "sell",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "kucoin", Currency: "USDC"},
					To:   models.AssetKey{Exchange: "kucoin", Currency: "ETH"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "ETH/USDC",
					From:          models.Asset{Exchange: "kucoin", Currency: "USDC", Index: 3},
					To:            models.Asset{Exchange: "kucoin", Currency: "ETH", Index: 2},
					Weight:        decimal.MustNew(2326859740901023, 19), // (1/4293.34) * (1 - 0.001)
					Side:          "buy",
				},
			},
		},
		{
			name:          "empty exchanges",
			exchangesPtr:  &models.Exchanges{},
			assetsPtr:     &models.Assets{},
			expectedPairs: models.Pairs{},
		},
		{
			name: "markets with zero bid or ask",
			exchangesPtr: &models.Exchanges{
				"binance": {
					Id: "binance",
					Markets: map[string]models.Market{
						"BTC/USDC": {
							Id:        "BTC/USDC",
							Base:      "BTC",
							Quote:     "USDC",
							Ask:       decimal.MustNew(11064601, 2), // 110646.01
							Bid:       decimal.MustNew(0, 0),        // 0
							TakerFee:  decimal.MustNew(1, 3),
							Timestamp: time.Now(),
						},
						"ETH/USDC": {
							Id:        "ETH/USDC",
							Base:      "ETH",
							Quote:     "USDC",
							Ask:       decimal.MustNew(0, 0),      // 0
							Bid:       decimal.MustNew(429300, 2), // 4293.00
							TakerFee:  decimal.MustNew(1, 3),
							Timestamp: time.Now(),
						},
					},
				},
			},
			assetsPtr: &models.Assets{
				models.AssetKey{Exchange: "binance", Currency: "BTC"}:  models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
				models.AssetKey{Exchange: "binance", Currency: "USDC"}: models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
				models.AssetKey{Exchange: "binance", Currency: "ETH"}:  models.Asset{Exchange: "binance", Currency: "ETH", Index: 2},
			},
			expectedPairs: models.Pairs{
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "USDC"},
					To:   models.AssetKey{Exchange: "binance", Currency: "BTC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "BTC/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					To:            models.Asset{Exchange: "binance", Currency: "BTC", Index: 0},
					Weight:        decimal.MustNew(90287937179117, 19),
					Side:          "buy",
				},
				models.PairKey{
					From: models.AssetKey{Exchange: "binance", Currency: "ETH"},
					To:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				}: models.Pair{
					IntraExchange: true,
					Symbol:        "ETH/USDC",
					From:          models.Asset{Exchange: "binance", Currency: "ETH", Index: 2},
					To:            models.Asset{Exchange: "binance", Currency: "USDC", Index: 1},
					Weight:        decimal.MustNew(428870700, 5),
					Side:          "sell",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPairs := createIntraExchangePairs(tc.exchangesPtr, tc.assetsPtr)

			// the order of pairs in the map can be non-deterministic due to concurrent workers.
			// compare the maps for equality.
			if !maps.EqualFunc(gotPairs, tc.expectedPairs, func(a, b models.Pair) bool {
				return a.IntraExchange == b.IntraExchange &&
					a.Symbol == b.Symbol &&
					a.From == b.From &&
					a.To == b.To &&
					a.Side == b.Side &&
					a.Network == b.Network &&
					a.Weight.Cmp(b.Weight) == 0
			}) {
				t.Fatalf("createIntraExchangePairs() returned incorrect pairs.\nGot: %+v\nWant: %+v", gotPairs, tc.expectedPairs)
			}
		})
	}
}
