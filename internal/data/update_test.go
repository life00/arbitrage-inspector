package data

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func newFloat64(f float64) *float64 {
	return &f
}

func newInt64(i int64) *int64 {
	return &i
}

func newString(s string) *string {
	return &s
}

func newBool(b bool) *bool {
	return &b
}

func TestUpdateExchange(t *testing.T) {
	testCases := []struct {
		name               string
		initialExchanges   models.Exchanges
		mockClient         *mockExchange
		updateCurrencyFees bool
		updateMarketFees   bool
		wantErr            bool
		wantErrContent     string
	}{
		{
			name: "Successful full update",
			initialExchanges: models.Exchanges{
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
			mockClient: &mockExchange{
				name: "testExchange",
				tickers: ccxt.Tickers{
					Tickers: map[string]ccxt.Ticker{
						"BTC/USDT": {
							Bid: newFloat64(50000),
						},
					},
				},
				apiCurrencies: ccxt.Currencies{
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
				markets: []ccxt.MarketInterface{
					{
						Symbol: newString("BTC/USDT"),
						Taker:  newFloat64(0.001),
					},
				},
			},
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            false,
			wantErrContent:     "",
		},
		{
			name: "Update prices only",
			initialExchanges: models.Exchanges{
				"testExchange": {
					Id: "testExchange",
					Markets: map[string]models.Market{
						"BTC/USDT": {},
					},
				},
			},
			mockClient: &mockExchange{
				name: "testExchange",
				tickers: ccxt.Tickers{
					Tickers: map[string]ccxt.Ticker{
						"BTC/USDT": {
							Bid: newFloat64(50000),
						},
					},
				},
			},
			updateCurrencyFees: false,
			updateMarketFees:   false,
			wantErr:            false,
			wantErrContent:     "",
		},
		{
			name: "Exchange not found",
			initialExchanges: models.Exchanges{
				"anotherExchange": {
					Id: "anotherExchange",
				},
			},
			mockClient: &mockExchange{
				name: "testExchange",
			},
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            true,
			wantErrContent:     "exchange not found in data structure",
		},
		{
			name: "Error in price update",
			initialExchanges: models.Exchanges{
				"testExchange": {
					Id: "testExchange",
					Markets: map[string]models.Market{
						"BTC/USDT": {},
					},
				},
			},
			mockClient: &mockExchange{
				name:              "testExchange",
				fetchTickersError: fmt.Errorf("something went wrong"),
			},
			updateCurrencyFees: true,
			updateMarketFees:   true,
			wantErr:            true,
			wantErrContent:     "API call failed: something went wrong",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mu := &sync.Mutex{}
			exchanges := &tc.initialExchanges
			var clientPtr ccxt.IExchange = tc.mockClient

			err := updateExchange(&clientPtr, mu, exchanges, tc.updateCurrencyFees, tc.updateMarketFees)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
				return
			}

			if tc.wantErr && err != nil {
				if err.Error() != tc.wantErrContent {
					t.Errorf("Expected error content: '%s', got: '%s'", tc.wantErrContent, err.Error())
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
		want        *models.Exchange
		wantErr     bool
	}{
		{
			name: "Successful update",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
					"ETH/SOL":  {Id: "ETH/SOL"},
				},
			},
			mockTickers: ccxt.Tickers{
				Tickers: map[string]ccxt.Ticker{
					"BTC/USDT": {
						Bid:       newFloat64(50000.50),
						Ask:       newFloat64(50001.00),
						Timestamp: newInt64(time.Now().UnixMilli()),
					},
					"ETH/SOL": {
						Bid:       newFloat64(3000.75),
						Ask:       newFloat64(3001.25),
						Timestamp: newInt64(time.Now().UnixMilli()),
					},
				},
			},
			want: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {
						Id:        "BTC/USDT",
						Bid:       decimal.Zero,
						Ask:       decimal.Zero,
						Timestamp: time.UnixMilli(time.Now().UnixMilli()),
					},
					"ETH/SOL": {
						Id:        "ETH/SOL",
						Bid:       decimal.Zero,
						Ask:       decimal.Zero,
						Timestamp: time.UnixMilli(time.Now().UnixMilli()),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty tickers",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
				},
			},
			mockTickers: ccxt.Tickers{Tickers: map[string]ccxt.Ticker{}},
			want:        &models.Exchange{Id: "testExchange", Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}}},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockExchange{tickers: tc.mockTickers}
			var clientPtr ccxt.IExchange = mockClient
			err := updatePrices(&clientPtr, tc.initial)

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
		want           *models.Exchange
		wantErr        bool
	}{
		{
			name: "Successful update with best network",
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
							"Bitcoin": {
								Active:   newBool(true),
								Fee:      newFloat64(0.0005),
								Deposit:  newBool(true),
								Withdraw: newBool(true),
							},
							"Lightning": {
								Active:   newBool(true),
								Fee:      newFloat64(0.0001),
								Deposit:  newBool(true),
								Withdraw: newBool(true),
							},
						},
					},
					"ETH": {
						Id: newString("ETH"),
						Networks: map[string]ccxt.Network{
							"ERC20": {
								Active:   newBool(true),
								Fee:      newFloat64(10),
								Deposit:  newBool(true),
								Withdraw: newBool(true),
							},
						},
					},
				},
			},
			want: &models.Exchange{
				Id: "testExchange",
				Currencies: map[string]models.Currency{
					"BTC": {
						Id:            "BTC",
						WithdrawalFee: decimal.Zero,
						Network:       "Lightning",
					},
					"ETH": {
						Id:            "ETH",
						WithdrawalFee: decimal.Zero,
						Network:       "ERC20",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty currencies",
			initial: &models.Exchange{
				Id: "testExchange",
				Currencies: map[string]models.Currency{
					"BTC": {Id: "BTC"},
				},
			},
			mockCurrencies: ccxt.Currencies{Currencies: map[string]ccxt.Currency{}},
			want:           &models.Exchange{Id: "testExchange", Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}}},
			wantErr:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockExchange{apiCurrencies: tc.mockCurrencies}
			var clientPtr ccxt.IExchange = mockClient
			err := updateCurrencies(&clientPtr, tc.initial)

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
		want        *models.Exchange
		wantErr     bool
	}{
		{
			name: "Successful update",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
					"ETH/SOL":  {Id: "ETH/SOL"},
				},
			},
			mockMarkets: []ccxt.MarketInterface{
				{
					Symbol: newString("BTC/USDT"),
					Taker:  newFloat64(0.001),
					Maker:  newFloat64(0.0005),
				},
				{
					Symbol: newString("ETH/SOL"),
					Taker:  newFloat64(0.002),
					Maker:  newFloat64(0.0015),
				},
			},
			want: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {
						Id:       "BTC-USDT",
						TakerFee: decimal.Zero,
						MakerFee: decimal.Zero,
					},
					"ETH/SOL": {
						Id:       "ETH-SOL",
						TakerFee: decimal.Zero,
						MakerFee: decimal.Zero,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty markets list",
			initial: &models.Exchange{
				Id: "testExchange",
				Markets: map[string]models.Market{
					"BTC/USDT": {Id: "BTC/USDT"},
				},
			},
			mockMarkets: []ccxt.MarketInterface{},
			want:        &models.Exchange{Id: "testExchange", Markets: map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT"}}},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockExchange{markets: tc.mockMarkets}
			var clientPtr ccxt.IExchange = mockClient
			err := updateMarkets(&clientPtr, tc.initial)

			if (err != nil) != tc.wantErr {
				t.Errorf("Expected error: %v, got: %v", tc.wantErr, err)
			}
		})
	}
}
