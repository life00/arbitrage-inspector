package data

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestValidateExchanges(t *testing.T) {
	testCases := []struct {
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateExchanges(tc.exchanges); (err != nil) != tc.wantErr {
				t.Errorf("validateExchanges() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestLoadCcxt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	testCases := []struct {
		name            string
		exchanges       []string
		wantErr         bool
		wantErrContains string
		wantLoaded      int
	}{
		{
			name:       "load valid public exchange",
			exchanges:  []string{"binance"},
			wantErr:    false,
			wantLoaded: 1,
		},
		{
			name:            "fail with invalid exchange name",
			exchanges:       []string{"invalidexchange"},
			wantErr:         true,
			wantErrContains: "failed to create CCXT exchange for invalidexchange",
			wantLoaded:      0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loadedExchanges, err := loadClient(tc.exchanges)

			if (err != nil) != tc.wantErr {
				t.Errorf("loadCcxt() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if err != nil && tc.wantErrContains != "" {
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("loadCcxt() error = %q, want error containing %q", err.Error(), tc.wantErrContains)
				}
			}

			if len(loadedExchanges) != tc.wantLoaded {
				t.Errorf("loadCcxt() loaded %d exchanges, want %d", len(loadedExchanges), tc.wantLoaded)
			}
		})
	}
}

func TestValidateCurrencies(t *testing.T) {
	testExchangeA := &TestExchange{
		Name:       "exchangeA",
		Currencies: []ccxt.Currency{newMockCurrency("BTC"), newMockCurrency("ETH")},
	}
	testExchangeB := &TestExchange{
		Name:       "exchangeB",
		Currencies: []ccxt.Currency{newMockCurrency("LTC"), newMockCurrency("ADA")},
	}
	testExchangeCWithInvalid := &TestExchange{
		Name: "exchangeC",
		Currencies: []ccxt.Currency{
			newMockCurrency("XRP"),
			{Id: newString("DOGE"), Active: newBool(false)},
			{Id: newString("SOL"), Deposit: newBool(false)},
			{Id: newString("DOT"), Withdraw: newBool(false)},
			{Id: nil},
		},
	}

	testCases := []struct {
		name            string
		testCurrencies  []string
		testClients     *models.Clients
		wantErr         bool
		wantErrContains string
	}{
		{
			name:           "valid currencies across multiple exchanges",
			testCurrencies: []string{"BTC", "LTC"},
			testClients:    &models.Clients{testExchangeA.Name: testExchangeA, testExchangeB.Name: testExchangeB},
			wantErr:        false,
		},
		{
			name:            "multiple currencies are invalid",
			testCurrencies:  []string{"BTC", "unsupported1", "unsupported2"},
			testClients:     &models.Clients{testExchangeA.Name: testExchangeA},
			wantErr:         true,
			wantErrContains: "invalid currencies: unsupported1, unsupported2",
		},
		{
			name:            "empty list of currencies to check",
			testCurrencies:  []string{},
			testClients:     &models.Clients{testExchangeA.Name: testExchangeA},
			wantErr:         true,
			wantErrContains: "list of currencies is empty",
		},
		{
			name:            "empty clients list",
			testCurrencies:  []string{"BTC"},
			testClients:     &models.Clients{},
			wantErr:         true,
			wantErrContains: "list of clients is empty",
		},
		{
			name:            "currency exists but is inactive",
			testCurrencies:  []string{"DOGE"},
			testClients:     &models.Clients{testExchangeCWithInvalid.Name: testExchangeCWithInvalid},
			wantErr:         true,
			wantErrContains: "invalid currencies: DOGE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCurrencies(tc.testCurrencies, tc.testClients)

			if (err != nil) != tc.wantErr {
				t.Errorf("validateCurrencies() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && err != nil {
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("validateCurrencies() error = %q, want error to contain %q", err.Error(), tc.wantErrContains)
				}
			}
		})
	}
}

func TestCreateData(t *testing.T) {
	testExchangeA := &TestExchange{
		Name: "exchangeA",
		Currencies: []ccxt.Currency{
			newMockCurrency("BTC"),
			newMockCurrency("ETH"),
			newMockCurrency("USDT"),
			func() ccxt.Currency {
				id, active := "XRP", false
				return ccxt.Currency{Id: &id, Active: &active}
			}(),
		},
		Markets: []ccxt.MarketInterface{
			newMockMarket("BTC/USDT", "BTC", "USDT", true, true),
			newMockMarket("ETH/DAI", "ETH", "DAI", true, true),
			newMockMarket("XRP/USDT", "XRP", "USDT", false, true),
			newMockMarket("LTC/USDT", "LTC", "USDT", true, false),
		},
	}
	testExchangeB := &TestExchange{
		Name: "exchangeB",
		Currencies: []ccxt.Currency{
			newMockCurrency("BTC"),
			newMockCurrency("USDT"),
			newMockCurrency("ADA"),
		},
		Markets: []ccxt.MarketInterface{
			newMockMarket("BTC/USDT", "BTC", "USDT", true, true),
			newMockMarket("ADA/USDT", "ADA", "USDT", true, true),
		},
	}

	testCases := []struct {
		name           string
		testCurrencies []string
		testClients    *models.Clients
		want           models.Exchanges
	}{
		{
			name:           "processes two exchanges concurrently",
			testCurrencies: []string{"BTC", "USDT", "ADA"},
			testClients:    &models.Clients{"exchangeA": testExchangeA, "exchangeB": testExchangeB},
			want: models.Exchanges{
				"exchangeA": {
					Id:         "exchangeA",
					Markets:    map[string]models.Market{"BTC/USDT": {Id: "BTC/USDT", Base: "BTC", Quote: "USDT"}},
					Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}, "USDT": {Id: "USDT"}},
				},
				"exchangeB": {
					Id:         "exchangeB",
					Markets:    map[string]models.Market{"ADA/USDT": {Id: "ADA/USDT", Base: "ADA", Quote: "USDT"}, "BTC/USDT": {Id: "BTC/USDT", Base: "BTC", Quote: "USDT"}},
					Currencies: map[string]models.Currency{"BTC": {Id: "BTC"}, "USDT": {Id: "USDT"}, "ADA": {Id: "ADA"}},
				},
			},
		},
		{
			name:           "handles no clients",
			testCurrencies: []string{"BTC", "USDT"},
			testClients:    &models.Clients{},
			want:           models.Exchanges{},
		},
		{
			name:           "handles nil clients pointer",
			testCurrencies: []string{"BTC", "USDT"},
			testClients:    nil,
			want:           models.Exchanges{},
		},
		{
			name:           "handles no input currencies",
			testCurrencies: []string{},
			testClients:    &models.Clients{"exchangeA": testExchangeA},
			want: models.Exchanges{
				"exchangeA": {
					Id:         "exchangeA",
					Markets:    map[string]models.Market{},
					Currencies: map[string]models.Currency{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := createExchanges(tc.testCurrencies, tc.testClients)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("createExchanges() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
