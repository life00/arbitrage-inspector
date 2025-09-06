package arbitrage

import (
	"testing"

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
