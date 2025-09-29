package fetch

import (
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestUpdateExchange(t *testing.T) {
	testCases := []struct {
		name               string
		testExchanges      models.Exchanges
		mockClient         *TestExchange
		updateCurrencyFees bool
		updateMarketFees   bool
		wantErr            bool
		wantErrContains    string
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
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            false,
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
			updateCurrencyFees: false,
			updateMarketFees:   false,
			wantErr:            false,
		},
		{
			name: "exchange not found",
			testExchanges: models.Exchanges{
				"anotherExchange": {Id: "anotherExchange"},
			},
			mockClient:         &TestExchange{Name: "testExchange"},
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            true,
			wantErrContains:    "exchange not found in data structure",
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
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            true,
			wantErrContains:    "API call failed: something went wrong",
		},
		{
			name: "error in currency fees update",
			testExchanges: models.Exchanges{
				"testExchange": {
					Id: "testExchange",
					Currencies: map[string]models.Currency{
						"BTC": {},
					},
				},
			},
			mockClient: &TestExchange{
				Name:                 "testExchange",
				FetchCurrenciesError: newMockError("currency fetch failed"),
			},
			updateCurrencyFees: true,
			updateMarketFees:   false,
			wantErr:            true,
			wantErrContains:    "API call failed: currency fetch failed",
		},
		{
			name: "error in market fees update",
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
				FetchMarketsError: newMockError("market fetch failed"),
			},
			updateCurrencyFees: false,
			updateMarketFees:   true,
			wantErr:            true,
			wantErrContains:    "API call failed: market fetch failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mu := &sync.Mutex{}
			exchanges := &tc.testExchanges
			var clientPtr ccxt.IExchange = tc.mockClient

			err := updateExchange(&clientPtr, mu, exchanges, tc.updateCurrencyFees, tc.updateMarketFees)

			if (err != nil) != tc.wantErr {
				t.Errorf("updateExchange() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && err != nil {
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("updateExchange() error = %q, want error containing %q", err.Error(), tc.wantErrContains)
				}
			}
		})
	}
}

// ---

func TestFetchPrices(t *testing.T) {
	testCases := []struct {
		name        string
		exchange    *models.Exchange
		mockTickers ccxt.Tickers
		wantErr     bool
	}{
		{
			name: "successful update",
			exchange: &models.Exchange{
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
			exchange: &models.Exchange{
				Id:      "testExchange",
				Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}},
			},
			mockTickers: ccxt.Tickers{Tickers: map[string]ccxt.Ticker{}},
			wantErr:     false,
		},
		{
			name: "invalid bid value",
			exchange: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {},
				},
			},
			mockTickers: ccxt.Tickers{
				Tickers: map[string]ccxt.Ticker{
					"BTC/USDT": {
						Symbol: newString("BTC/USDT"),
						Bid:    newFloat64(math.Inf(1)),
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{Tickers: tc.mockTickers}
			var clientPtr ccxt.IExchange = mockClient
			updatedPrices, err := fetchPrices(&clientPtr, tc.exchange)

			if (err != nil) != tc.wantErr {
				t.Errorf("fetchPrices() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && updatedPrices != nil {
				// Assert that the returned map contains the correct data
				if len(updatedPrices) != len(tc.mockTickers.Tickers) {
					t.Errorf("fetchPrices() loaded %d prices, want %d", len(updatedPrices), len(tc.mockTickers.Tickers))
				}
				for id, ticker := range tc.mockTickers.Tickers {
					if updated, ok := updatedPrices[id]; ok {
						expectedBid, _ := decimal.NewFromFloat64(*ticker.Bid)
						expectedAsk, _ := decimal.NewFromFloat64(*ticker.Ask)
						if !updated.Bid.Equal(expectedBid) || !updated.Ask.Equal(expectedAsk) {
							t.Errorf("Mismatch for %s. Expected bid: %v, ask: %v. Got bid: %v, ask: %v", id, expectedBid, expectedAsk, updated.Bid, updated.Ask)
						}
						expectedTimestamp := time.UnixMilli(*ticker.Timestamp)
						if !updated.Timestamp.Equal(expectedTimestamp) {
							t.Errorf("Mismatch for %s timestamp. Expected: %v, got: %v", id, expectedTimestamp, updated.Timestamp)
						}
					} else {
						t.Errorf("Expected to find %s in updated prices", id)
					}
				}
			}
		})
	}
}

// ---

func TestFetchCurrencies(t *testing.T) {
	testCases := []struct {
		name           string
		exchange       *models.Exchange
		mockCurrencies ccxt.Currencies
		wantErr        bool
	}{
		{
			name: "successful update with best network",
			exchange: &models.Exchange{
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
							"BITCOIN":   {Active: newBool(true), Fee: newFloat64(0.0005), Deposit: newBool(true), Withdraw: newBool(true)},
							"LIGHTNING": {Active: newBool(true), Fee: newFloat64(0.0001), Deposit: newBool(true), Withdraw: newBool(true)},
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
			exchange: &models.Exchange{
				Id:         "testExchange",
				Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}},
			},
			mockCurrencies: ccxt.Currencies{Currencies: map[string]ccxt.Currency{}},
			wantErr:        false,
		},
		{
			name: "invalid fee value",
			exchange: &models.Exchange{
				Id:         "testExchange",
				Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}},
			},
			mockCurrencies: ccxt.Currencies{
				Currencies: map[string]ccxt.Currency{
					"BTC": {
						Id: newString("BTC"),
						Networks: map[string]ccxt.Network{
							"Bitcoin": {Active: newBool(true), Fee: newFloat64(math.Inf(1)), Deposit: newBool(true), Withdraw: newBool(true)},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{APICurrencies: tc.mockCurrencies}
			var clientPtr ccxt.IExchange = mockClient
			updatedCurrencies, err := fetchCurrencies(&clientPtr, tc.exchange)

			if (err != nil) != tc.wantErr {
				t.Errorf("fetchCurrencies() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && updatedCurrencies != nil {
				for id, mockCurrency := range tc.mockCurrencies.Currencies {
					if updated, ok := updatedCurrencies[id]; ok {
						expectedNetworks := 0
						for _, network := range mockCurrency.Networks {
							if *network.Fee != math.Inf(1) {
								expectedNetworks++
							}
						}
						if len(updated.Networks) != expectedNetworks {
							t.Errorf("fetchCurrencies() loaded %d networks, want %d", len(updated.Networks), expectedNetworks)
						}

						for name, network := range mockCurrency.Networks {
							if *network.Fee != math.Inf(1) {
								if updatedNetwork, ok := updated.Networks[name]; ok {
									expectedFee, _ := decimal.NewFromFloat64(*network.Fee)
									if !updatedNetwork.WithdrawalFee.Equal(expectedFee) {
										t.Errorf("Mismatch for network %s. Expected fee: %v, got: %v", name, expectedFee, updatedNetwork.WithdrawalFee)
									}
								} else {
									t.Errorf("Expected to find network %s in updated currencies", name)
								}
							}
						}
					} else {
						t.Errorf("Expected to find %s in updated currencies", id)
					}
				}
			}
		})
	}
}

// ---

func TestFetchMarkets(t *testing.T) {
	testCases := []struct {
		name        string
		exchange    *models.Exchange
		mockMarkets []ccxt.MarketInterface
		wantErr     bool
	}{
		{
			name: "successful update",
			exchange: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
					"ETH/SOL":  {Id: "ETH/SOL"},
				},
			},
			mockMarkets: []ccxt.MarketInterface{
				{Symbol: newString("BTC/USDT"), Taker: newFloat64(0.001)},
				{Symbol: newString("ETH/SOL"), Taker: newFloat64(0.002)},
			},
			wantErr: false,
		},
		{
			name: "empty markets list",
			exchange: &models.Exchange{
				Id:      "testExchange",
				Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}},
			},
			mockMarkets: []ccxt.MarketInterface{},
			wantErr:     false,
		},
		{
			name: "invalid taker fee",
			exchange: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {},
				},
			},
			mockMarkets: []ccxt.MarketInterface{
				{Symbol: newString("BTC/USDT"), Taker: newFloat64(math.Inf(1))},
			},
			wantErr: true,
		},
		{
			name: "invalid taker fee",
			exchange: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {},
				},
			},
			mockMarkets: []ccxt.MarketInterface{
				{Symbol: newString("BTC/USDT"), Taker: newFloat64(math.Inf(1))},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &TestExchange{Markets: tc.mockMarkets}
			var clientPtr ccxt.IExchange = mockClient
			updatedMarkets, err := fetchFees(&clientPtr, tc.exchange)

			if (err != nil) != tc.wantErr {
				t.Errorf("fetchFees() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && updatedMarkets != nil {
				for _, mockMarket := range tc.mockMarkets {
					if updated, ok := updatedMarkets[*mockMarket.Symbol]; ok {
						expectedTaker, _ := decimal.NewFromFloat64(*mockMarket.Taker)
						if !updated.TakerFee.Equal(expectedTaker) {
							t.Errorf("fetchFees() mismatch for %s: want taker fee: %v, got taker fee: %v", *mockMarket.Symbol, expectedTaker, updated.TakerFee)
						}
					}
				}
			}
		})
	}
}
