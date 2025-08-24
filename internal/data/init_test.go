package data

import (
	"log/slog"
	"os"
	// "reflect"
	// "sort"
	"strings"
	"testing"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestMain(m *testing.M) {
	// load .env for API credentials
	err := godotenv.Load("../../.env")
	if err != nil {
		slog.Error("failed to load .env file")
		os.Exit(1)
	}

	m.Run()
	os.Exit(0)
}

func TestValidateExchanges(t *testing.T) {
	tests := []struct {
		name      string
		exchanges []string
		wantErr   bool
	}{
		{
			name:      "valid exchanges",
			exchanges: []string{"binance", "kucoin"},
			wantErr:   false,
		},
		{
			name:      "invalid exchanges",
			exchanges: []string{"invalidexchange"},
			wantErr:   true,
		},
		{
			name:      "mixed valid and invalid exchanges",
			exchanges: []string{"binance", "invalidexchange"},
			wantErr:   true,
		},
		{
			name:      "empty exchanges",
			exchanges: []string{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateExchanges(tt.exchanges); (err != nil) != tt.wantErr {
				t.Errorf("validateExchanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadCcxt(t *testing.T) {
	// Skip integration tests if the -short flag is provided, as they make network calls.
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	tests := []struct {
		name        string
		exchanges   []string
		wantErr     bool
		errContains string
		wantLoaded  int
	}{
		{
			name:       "load valid public exchange (integration)",
			exchanges:  []string{"binance"},
			wantErr:    false,
			wantLoaded: 1,
		},
		{
			name:        "fail with invalid exchange name",
			exchanges:   []string{"invalidexchange"},
			wantErr:     true,
			errContains: "failed to create CCXT exchange for invalidexchange",
			wantLoaded:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadedExchanges, err := loadClient(tt.exchanges)

			if (err != nil) != tt.wantErr {
				t.Errorf("loadCcxt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("loadCcxt() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			}

			if len(loadedExchanges) != tt.wantLoaded {
				t.Errorf("loadCcxt() loaded %d exchanges, want %d", len(loadedExchanges), tt.wantLoaded)
			}
		})
	}
}

// mockExchange implements the ccxt.IExchange interface for testing.
// we only need to implement the methods that getCommonCurrencies actually calls.
type mockExchange struct {
	ccxt.IExchange // Embed the interface to satisfy it implicitly.
	name           string
	currencies     []ccxt.Currency
	markets        []ccxt.MarketInterface
}

// GetCurrenciesList overrides the embedded interface's method.
func (m *mockExchange) GetCurrenciesList() []ccxt.Currency {
	return m.currencies
}

// newCurrency is a test helper to create a ccxt.Currency
// with a non-nil ID pointer and all boolean fields set to true.
func newCurrency(id string) ccxt.Currency {
	active := true
	deposit := true
	withdraw := true
	return ccxt.Currency{
		Id:       &id,
		Active:   &active,
		Deposit:  &deposit,
		Withdraw: &withdraw,
	}
}

func TestValidateCurrencies(t *testing.T) {
	exchangeA := &mockExchange{
		name:       "exchangeA",
		currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("ETH")},
	}

	exchangeB := &mockExchange{
		name:       "exchangeB",
		currencies: []ccxt.Currency{newCurrency("LTC"), newCurrency("ADA")},
	}

	activeTrue, inactiveFalse := true, false
	idDoge, idSol, idDot := "DOGE", "SOL", "DOT"

	exchangeCWithInvalid := &mockExchange{
		name: "exchangeC",
		currencies: []ccxt.Currency{
			newCurrency("XRP"), // This one is valid
			{Id: &idDoge, Active: &inactiveFalse, Deposit: &activeTrue, Withdraw: &activeTrue}, // Inactive
			{Id: &idSol, Active: &activeTrue, Deposit: &inactiveFalse, Withdraw: &activeTrue},  // Deposit disabled
			{Id: &idDot, Active: &activeTrue, Deposit: &activeTrue, Withdraw: &inactiveFalse},  // Withdraw disabled
			{Id: nil, Active: &activeTrue, Deposit: &activeTrue, Withdraw: &activeTrue},        // Nil ID
		},
	}

	tests := []struct {
		name        string
		currencies  []string
		clientsPtr  *models.Clients
		wantErr     bool
		errContains string // For checking the error message content
	}{
		{
			name:       "valid currencies across multiple exchanges",
			currencies: []string{"BTC", "LTC"},
			clientsPtr: &models.Clients{exchangeA.name: exchangeA, exchangeB.name: exchangeB},
			wantErr:    false,
		},
		{
			name:        "multiple currencies are invalid",
			currencies:  []string{"BTC", "unsupported1", "unsupported2"},
			clientsPtr:  &models.Clients{exchangeA.name: exchangeA},
			wantErr:     true,
			errContains: "invalid currencies: unsupported1, unsupported2",
		},
		{
			name:        "empty list of currencies to check",
			currencies:  []string{},
			clientsPtr:  &models.Clients{exchangeA.name: exchangeA},
			wantErr:     true,
			errContains: "list of currencies is empty",
		},
		{
			name:        "empty clients list",
			currencies:  []string{"BTC"},
			clientsPtr:  &models.Clients{},
			wantErr:     true,
			errContains: "list of clients is empty",
		},
		{
			name:        "currency exists but is inactive",
			currencies:  []string{"DOGE"},
			clientsPtr:  &models.Clients{exchangeCWithInvalid.name: exchangeCWithInvalid},
			wantErr:     true,
			errContains: "invalid currencies: DOGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCurrencies(tt.currencies, tt.clientsPtr)

			if !tt.wantErr && err != nil {
				t.Errorf("validateCurrencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err == nil {
				t.Errorf("validateCurrencies() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateCurrencies() error = %q, want error to contain %q", err.Error(), tt.errContains)
				}
			}
		})
	}
}

// // GetMarketsList overrides the embedded interface's method.
// func (m *mockExchange) GetMarketsList() []ccxt.MarketInterface {
// 	return m.markets
// }
//
// // newMarket is a test helper to create a ccxt.Market with all required fields valid.
// func newMarket(symbol, baseId, quoteId string) ccxt.MarketInterface {
// 	active := true
// 	spot := true
// 	return ccxt.MarketInterface{
// 		Symbol:  &symbol,
// 		BaseId:  &baseId,
// 		QuoteId: &quoteId,
// 		Active:  &active,
// 		Spot:    &spot,
// 	}
// }
//
// func TestGetCommonValidMarkets(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		exchanges []ccxt.IExchange
// 		want      models.Markets
// 	}{
// 		{
// 			name:      "no exchanges",
// 			exchanges: []ccxt.IExchange{},
// 			want:      models.Markets{},
// 		},
// 		{
// 			name: "one exchange",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{
// 					name: "exchangeA",
// 					markets: []ccxt.MarketInterface{
// 						newMarket("BTC/USDC", "BTC", "USDC"),
// 						newMarket("ETH/USDC", "ETH", "USDC"),
// 					},
// 				},
// 			},
// 			want: models.Markets{Markets: []models.Market{{Id: "BTC/USDC", Base: "BTC", Quote: "USDC"}, {Id: "ETH/USDC", Base: "ETH", Quote: "USDC"}}},
// 		},
// 		{
// 			name: "two exchanges with common markets",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{
// 					name: "exchangeA",
// 					markets: []ccxt.MarketInterface{
// 						newMarket("BTC/USDC", "BTC", "USDC"),
// 						newMarket("ETH/USDC", "ETH", "USDC"),
// 						newMarket("XRP/USDC", "XRP", "USDC"),
// 					},
// 				},
// 				&mockExchange{
// 					name: "exchangeB",
// 					markets: []ccxt.MarketInterface{
// 						newMarket("ETH/USDC", "ETH", "USDC"),
// 						newMarket("LTC/USDC", "LTC", "USDC"),
// 						newMarket("BTC/USDC", "BTC", "USDC"),
// 					},
// 				},
// 			},
// 			want: models.Markets{Markets: []models.Market{{Id: "BTC/USDC", Base: "BTC", Quote: "USDC"}, {Id: "ETH/USDC", Base: "ETH", Quote: "USDC"}}},
// 		},
// 		{
// 			name: "multiple exchanges with one common market",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{name: "A", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), newMarket("ETH/USDC", "ETH", "USDC")}},
// 				&mockExchange{name: "B", markets: []ccxt.MarketInterface{newMarket("LTC/USDC", "LTC", "USDC"), newMarket("BTC/USDC", "BTC", "USDC")}},
// 				&mockExchange{name: "C", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), newMarket("XRP/USDC", "XRP", "USDC")}},
// 			},
// 			want: models.Markets{Markets: []models.Market{{Id: "BTC/USDC", Base: "BTC", Quote: "USDC"}}},
// 		},
// 		{
// 			name: "exchanges with no common markets",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{name: "A", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), newMarket("ETH/USDC", "ETH", "USDC")}},
// 				&mockExchange{name: "B", markets: []ccxt.MarketInterface{newMarket("LTC/USDC", "LTC", "USDC"), newMarket("XRP/USDC", "XRP", "USDC")}},
// 			},
// 			want: models.Markets{Markets: []models.Market{}},
// 		},
// 		{
// 			name: "handles markets with nil ID gracefully",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{name: "A", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), {Symbol: nil}, newMarket("ETH/USDC", "ETH", "USDC")}},
// 				&mockExchange{name: "B", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), newMarket("LTC/USDC", "LTC", "USDC")}},
// 			},
// 			want: models.Markets{Markets: []models.Market{{Id: "BTC/USDC", Base: "BTC", Quote: "USDC"}}},
// 		},
// 		{
// 			name: "handles inactive markets",
// 			exchanges: []ccxt.IExchange{
// 				&mockExchange{name: "A", markets: []ccxt.MarketInterface{
// 					newMarket("BTC/USDC", "BTC", "USDC"),
// 					func() ccxt.MarketInterface {
// 						symbol, active, spot := "ETH/USDC", false, true
// 						return ccxt.MarketInterface{Symbol: &symbol, Active: &active, Spot: &spot}
// 					}(),
// 				}},
// 				&mockExchange{name: "B", markets: []ccxt.MarketInterface{newMarket("BTC/USDC", "BTC", "USDC"), newMarket("LTC/USDC", "LTC", "USDC")}},
// 			},
// 			want: models.Markets{Markets: []models.Market{{Id: "BTC/USDC", Base: "BTC", Quote: "USDC"}}},
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			got := getCommonValidMarkets(&tt.exchanges)
//
// 			// Sort both slices for consistent comparison.
// 			sort.Slice(got.Markets, func(i, j int) bool {
// 				return got.Markets[i].Id < got.Markets[j].Id
// 			})
// 			sort.Slice(tt.want.Markets, func(i, j int) bool {
// 				return tt.want.Markets[i].Id < tt.want.Markets[j].Id
// 			})
//
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("getCommonValidMarkets() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestGetMatchingMarkets(t *testing.T) {
// 	testCases := []struct {
// 		name          string
// 		commonMarkets models.Markets
// 		currencies    models.Currencies
// 		want          models.Markets
// 	}{
// 		{
// 			name: "All markets match",
// 			commonMarkets: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 					{Id: "2", Base: "BTC", Quote: "USD"},
// 				},
// 			},
// 			currencies: models.Currencies{
// 				Currencies: []models.Currency{
// 					{Id: "USD"},
// 					{Id: "EUR"},
// 					{Id: "BTC"},
// 				},
// 			},
// 			want: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 					{Id: "2", Base: "BTC", Quote: "USD"},
// 				},
// 			},
// 		},
// 		{
// 			name: "Some markets match",
// 			commonMarkets: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 					{Id: "2", Base: "BTC", Quote: "USD"},
// 					{Id: "3", Base: "JPY", Quote: "EUR"}, // Base JPY is not in currencies list
// 				},
// 			},
// 			currencies: models.Currencies{
// 				Currencies: []models.Currency{
// 					{Id: "USD"},
// 					{Id: "EUR"},
// 					{Id: "BTC"},
// 				},
// 			},
// 			want: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 					{Id: "2", Base: "BTC", Quote: "USD"},
// 				},
// 			},
// 		},
// 		{
// 			name: "No markets match",
// 			commonMarkets: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 					{Id: "2", Base: "BTC", Quote: "USD"},
// 				},
// 			},
// 			currencies: models.Currencies{
// 				Currencies: []models.Currency{
// 					{Id: "AUD"},
// 					{Id: "CAD"},
// 				},
// 			},
// 			want: models.Markets{
// 				Markets: nil,
// 			},
// 		},
// 		{
// 			name: "Empty common markets",
// 			commonMarkets: models.Markets{
// 				Markets: []models.Market{},
// 			},
// 			currencies: models.Currencies{
// 				Currencies: []models.Currency{
// 					{Id: "USD"},
// 				},
// 			},
// 			want: models.Markets{
// 				Markets: nil,
// 			},
// 		},
// 		{
// 			name: "Empty currencies list",
// 			commonMarkets: models.Markets{
// 				Markets: []models.Market{
// 					{Id: "1", Base: "USD", Quote: "EUR"},
// 				},
// 			},
// 			currencies: models.Currencies{
// 				Currencies: []models.Currency{},
// 			},
// 			want: models.Markets{
// 				Markets: nil,
// 			},
// 		},
// 	}
//
// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			result := getMatchingMarkets(tc.commonMarkets, tc.currencies)
// 			if !reflect.DeepEqual(result, tc.want) {
// 				t.Errorf("For test case '%s', got unexpected result.\nExpected: %+v\nGot: %+v", tc.name, tc.want, result)
// 			}
// 		})
// 	}
// }
