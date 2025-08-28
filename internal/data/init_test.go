package data

import (
	"log/slog"
	"os"
	"reflect"
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
type mockExchange struct {
	ccxt.IExchange // Embed the interface to satisfy it implicitly.
	name           string
	currencies     []ccxt.Currency
	markets        []ccxt.MarketInterface
}

// GetId overrides the embedded interface's method.
func (m *mockExchange) GetId() string {
	return m.name
}

// GetCurrenciesList overrides the embedded interface's method.
func (m *mockExchange) GetCurrenciesList() []ccxt.Currency {
	return m.currencies
}

// GetMarketsList overrides the embedded interface's method.
func (m *mockExchange) GetMarketsList() []ccxt.MarketInterface {
	return m.markets
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

// newMarket is a test helper to create a ccxt.MarketInterface with all required fields valid.
func newMarket(symbol, baseId, quoteId string, active, spot bool) ccxt.MarketInterface {
	return ccxt.MarketInterface{
		Symbol:  &symbol,
		BaseId:  &baseId,
		QuoteId: &quoteId,
		Active:  &active,
		Spot:    &spot,
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

func TestCreateData(t *testing.T) {
	exchangeA := &mockExchange{
		name: "exchangeA",
		currencies: []ccxt.Currency{
			newCurrency("BTC"),
			newCurrency("ETH"),
			newCurrency("USDT"),
			// Inactive currency
			func() ccxt.Currency {
				id, active := "XRP", false
				return ccxt.Currency{Id: &id, Active: &active}
			}(),
		},
		markets: []ccxt.MarketInterface{
			// Valid market
			newMarket("BTC/USDT", "BTC", "USDT", true, true),
			// Market with a currency not in the input set
			newMarket("ETH/DAI", "ETH", "DAI", true, true),
			// Inactive market
			newMarket("XRP/USDT", "XRP", "USDT", false, true),
			// Non-spot market
			newMarket("LTC/USDT", "LTC", "USDT", true, false),
		},
	}

	exchangeB := &mockExchange{
		name: "exchangeB",
		currencies: []ccxt.Currency{
			newCurrency("BTC"),
			newCurrency("USDT"),
			newCurrency("ADA"),
		},
		markets: []ccxt.MarketInterface{
			newMarket("BTC/USDT", "BTC", "USDT", true, true),
			newMarket("ADA/USDT", "ADA", "USDT", true, true),
		},
	}

	tests := []struct {
		name       string
		currencies []string
		clientsPtr *models.Clients
		want       models.Exchanges
	}{
		{
			name:       "Processes two exchanges concurrently",
			currencies: []string{"BTC", "USDT", "ADA"},
			clientsPtr: &models.Clients{
				"exchangeA": exchangeA,
				"exchangeB": exchangeB,
			},
			want: models.Exchanges{
				"exchangeA": {
					Id: "exchangeA",
					// Changed to a map
					Markets: map[string]models.Market{
						"BTC/USDT": {Id: "BTC/USDT", Base: "BTC", Quote: "USDT"},
					},
					Currencies: map[string]models.Currency{
						"BTC":  {Id: "BTC"},
						"USDT": {Id: "USDT"},
						"ADA":  {Id: "ADA"},
					},
				},
				"exchangeB": {
					Id: "exchangeB",
					// Changed to a map
					Markets: map[string]models.Market{
						"ADA/USDT": {Id: "ADA/USDT", Base: "ADA", Quote: "USDT"},
						"BTC/USDT": {Id: "BTC/USDT", Base: "BTC", Quote: "USDT"},
					},
					Currencies: map[string]models.Currency{
						"BTC":  {Id: "BTC"},
						"USDT": {Id: "USDT"},
						"ADA":  {Id: "ADA"},
					},
				},
			},
		},
		{
			name:       "Handles no clients",
			currencies: []string{"BTC", "USDT"},
			clientsPtr: &models.Clients{},
			want:       models.Exchanges{},
		},
		{
			name:       "Handles nil clients pointer",
			currencies: []string{"BTC", "USDT"},
			clientsPtr: nil,
			want:       models.Exchanges{},
		},
		{
			name:       "Handles no input currencies",
			currencies: []string{},
			clientsPtr: &models.Clients{"exchangeA": exchangeA},
			want: models.Exchanges{
				"exchangeA": {
					Id: "exchangeA",
					// Changed to an empty map
					Markets:    map[string]models.Market{},
					Currencies: map[string]models.Currency{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createData(tt.currencies, tt.clientsPtr)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createData() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
