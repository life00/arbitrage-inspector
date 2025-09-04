package data

import (
	"sync"
	"testing"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestUpdateData(t *testing.T) {
	testCases := []struct {
		name            string
		testExchanges   models.Exchanges
		mockClient      *TestExchange
		updateFees      bool
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "successful full update",
			testExchanges: models.Exchanges{
				"testExchange": {
					Id: "testExchange",
					Markets: map[string]models.Market{
						"BTC/USDT": {},
					},
					Currencies: map[string]models.Currency{
						"BTC": {},
					},
				},
			},
			mockClient: &TestExchange{
				Name: "testExchange",
				Tickers: ccxt.Tickers{
					Tickers: map[string]ccxt.Ticker{
						"BTC/USDT": newMockTicker("BTC/USDT", 50000, 50001),
					},
				},
				APICurrencies: ccxt.Currencies{
					Currencies: map[string]ccxt.Currency{
						"BTC": {
							Networks: map[string]ccxt.Network{
								"testnet": {
									Active:   newBool(true),
									Fee:      newFloat64(0.0001),
									Withdraw: newBool(true),
									Deposit:  newBool(true),
								},
							},
						},
					},
				},
				Markets: []ccxt.MarketInterface{
					{Symbol: newString("BTC/USDT"), Taker: newFloat64(0.001)},
				},
			},
			updateFees:      true,
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "update prices only",
			testExchanges: models.Exchanges{
				"testExchange": {
					Id:      "testExchange",
					Markets: map[string]models.Market{"BTC/USDT": {}},
				},
			},
			mockClient: &TestExchange{
				Name: "testExchange",
				Tickers: ccxt.Tickers{
					Tickers: map[string]ccxt.Ticker{
						"BTC/USDT": newMockTicker("BTC/USDT", 50000, 50001),
					},
				},
			},
			updateFees:      false,
			wantErr:         false,
			wantErrContains: "",
		},
		{
			name: "exchange not found",
			testExchanges: models.Exchanges{
				"anotherExchange": {Id: "anotherExchange"},
			},
			mockClient:      &TestExchange{Name: "testExchange"},
			updateFees:      true,
			wantErr:         true,
			wantErrContains: "exchange not found in data structure",
		},
		{
			name: "error in price update",
			testExchanges: models.Exchanges{
				"testExchange": {
					Id: "testExchange",
					Markets: map[string]models.Market{
						"BTC/USDT": {},
					},
				},
			},
			mockClient: &TestExchange{
				Name:              "testExchange",
				FetchTickersError: newMockError("something went wrong"),
			},
			updateFees:      true,
			wantErr:         true,
			wantErrContains: "API call failed: something went wrong",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mu := &sync.Mutex{}
			exchanges := &tc.testExchanges
			var clientPtr ccxt.IExchange = tc.mockClient

			err := updateExchange(&clientPtr, mu, exchanges, tc.updateFees, tc.updateFees)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
				return
			}
			if tc.wantErr && err != nil {
				if err.Error() != tc.wantErrContains {
					t.Errorf("Expected error content: '%s', got: '%s'", tc.wantErrContains, err.Error())
				}
			}
		})
	}
}

func TestUpdatePrices(t *testing.T) {
	testCases := []struct {
		name        string
		initial     *models.Exchange
		mockTickers ccxt.Tickers
		wantErr     bool
	}{
		{
			name: "successful update",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
					"ETH/SOL":  {Id: "ETH/SOL"},
				},
			},
			mockTickers: ccxt.Tickers{
				Tickers: map[string]ccxt.Ticker{
					"BTC/USDT": newMockTicker("BTC/USDT", 50000.50, 50001.00),
					"ETH/SOL":  newMockTicker("ETH/SOL", 3000.75, 3001.25),
				},
			},
			wantErr: false,
		},
		{
			name: "empty tickers",
			initial: &models.Exchange{
				Id:      "testExchange",
				Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}},
			},
			mockTickers: ccxt.Tickers{Tickers: map[string]ccxt.Ticker{}},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{Tickers: tc.mockTickers}
			var clientPtr ccxt.IExchange = mockClient
			var exchangeMu sync.Mutex
			err := updatePrices(&clientPtr, tc.initial, &exchangeMu)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestUpdateCurrencies(t *testing.T) {
	testCases := []struct {
		name           string
		initial        *models.Exchange
		mockCurrencies ccxt.Currencies
		wantErr        bool
	}{
		{
			name: "successful update with best network",
			initial: &models.Exchange{
				Id: "testExchange",
				Currencies: map[string]models.Currency{
					"BTC": {Id: "BTC"},
					"ETH": {Id: "ETH"},
				},
			},
			mockCurrencies: ccxt.Currencies{
				Currencies: map[string]ccxt.Currency{
					"BTC": {
						Id: newString("BTC"),
						Networks: map[string]ccxt.Network{
							"Bitcoin":   {Active: newBool(true), Fee: newFloat64(0.0005), Deposit: newBool(true), Withdraw: newBool(true)},
							"Lightning": {Active: newBool(true), Fee: newFloat64(0.0001), Deposit: newBool(true), Withdraw: newBool(true)},
						},
					},
					"ETH": {
						Id: newString("ETH"),
						Networks: map[string]ccxt.Network{
							"ERC20": {Active: newBool(true), Fee: newFloat64(10), Deposit: newBool(true), Withdraw: newBool(true)},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty currencies",
			initial: &models.Exchange{
				Id:         "testExchange",
				Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}},
			},
			mockCurrencies: ccxt.Currencies{Currencies: map[string]ccxt.Currency{}},
			wantErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{APICurrencies: tc.mockCurrencies}
			var clientPtr ccxt.IExchange = mockClient
			var exchangeMu sync.Mutex
			err := updateCurrencies(&clientPtr, tc.initial, &exchangeMu)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestUpdateMarkets(t *testing.T) {
	testCases := []struct {
		name        string
		initial     *models.Exchange
		mockMarkets []ccxt.MarketInterface
		wantErr     bool
	}{
		{
			name: "successful update",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
					"ETH/SOL":  {Id: "ETH/SOL"},
				},
			},
			mockMarkets: []ccxt.MarketInterface{
				{Symbol: newString("BTC/USDT"), Taker: newFloat64(0.001), Maker: newFloat64(0.0005)},
				{Symbol: newString("ETH/SOL"), Taker: newFloat64(0.002), Maker: newFloat64(0.0015)},
			},
			wantErr: false,
		},
		{
			name: "empty markets list",
			initial: &models.Exchange{
				Id:      "testExchange",
				Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}},
			},
			mockMarkets: []ccxt.MarketInterface{},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{Markets: tc.mockMarkets}
			var clientPtr ccxt.IExchange = mockClient
			var exchangeMu sync.Mutex
			err := updateMarkets(&clientPtr, tc.initial, &exchangeMu)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
			}
		})
	}
}
