package data

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestValidateExchanges(t *testing.T) {
	tests := []struct {
		name      string
		exchanges models.Exchanges
		wantErr   bool
	}{
		{
			name: "valid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "binance"},
					{Name: "kucoin"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "invalidexchange"},
				},
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "binance"},
					{Name: "invalidexchange"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{},
			},
			wantErr: false,
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
		exchanges   models.Exchanges
		wantErr     bool
		errContains string
		wantLoaded  int
	}{
		{
			name: "load valid public exchange (integration)",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{{Name: "binance"}},
			},
			wantErr:    false,
			wantLoaded: 1,
		},
		{
			name: "fail with invalid exchange name",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{{Name: "nonexistentexchange123"}},
			},
			wantErr:     true,
			errContains: "failed to create CCXT exchange for nonexistentexchange123",
			wantLoaded:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadedExchanges, err := loadCcxt(tt.exchanges)

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
}

// GetCurrenciesList overrides the embedded interface's method.
func (m *mockExchange) GetCurrenciesList() []ccxt.Currency {
	return m.currencies
}

// newCurrency is a test helper to create a ccxt.Currency with a non-nil ID pointer.
func newCurrency(id string) ccxt.Currency {
	return ccxt.Currency{Id: &id}
}

func TestGetCommonCurrencies(t *testing.T) {
	tests := []struct {
		name      string
		exchanges []ccxt.IExchange
		want      models.Currencies
	}{
		{
			name:      "no exchanges",
			exchanges: []ccxt.IExchange{},
			want:      models.Currencies{},
		},
		{
			name: "one exchange",
			exchanges: []ccxt.IExchange{
				&mockExchange{
					name:       "exchangeA",
					currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("ETH")},
				},
			},
			want: models.Currencies{Currencies: []models.Currency{{Code: "BTC"}, {Code: "ETH"}}},
		},
		{
			name: "two exchanges with common currencies",
			exchanges: []ccxt.IExchange{
				&mockExchange{
					name:       "exchangeA",
					currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("ETH"), newCurrency("XRP")},
				},
				&mockExchange{
					name:       "exchangeB",
					currencies: []ccxt.Currency{newCurrency("ETH"), newCurrency("LTC"), newCurrency("BTC")},
				},
			},
			want: models.Currencies{Currencies: []models.Currency{{Code: "BTC"}, {Code: "ETH"}}},
		},
		{
			name: "multiple exchanges with one common currency",
			exchanges: []ccxt.IExchange{
				&mockExchange{name: "A", currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("ETH")}},
				&mockExchange{name: "B", currencies: []ccxt.Currency{newCurrency("LTC"), newCurrency("BTC")}},
				&mockExchange{name: "C", currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("XRP")}},
			},
			want: models.Currencies{Currencies: []models.Currency{{Code: "BTC"}}},
		},
		{
			name: "exchanges with no common currencies",
			exchanges: []ccxt.IExchange{
				&mockExchange{name: "A", currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("ETH")}},
				&mockExchange{name: "B", currencies: []ccxt.Currency{newCurrency("LTC"), newCurrency("XRP")}},
			},
			want: models.Currencies{Currencies: []models.Currency{}}, // Expect empty slice, not nil
		},
		{
			name: "handles currencies with nil ID gracefully",
			exchanges: []ccxt.IExchange{
				&mockExchange{name: "A", currencies: []ccxt.Currency{newCurrency("BTC"), {Id: nil}, newCurrency("ETH")}},
				&mockExchange{name: "B", currencies: []ccxt.Currency{newCurrency("BTC"), newCurrency("LTC")}},
			},
			want: models.Currencies{Currencies: []models.Currency{{Code: "BTC"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCommonCurrencies(&tt.exchanges)

			// Sort both slices for consistent comparison, as map iteration order is not guaranteed.
			sort.Slice(got.Currencies, func(i, j int) bool {
				return got.Currencies[i].Code < got.Currencies[j].Code
			})
			sort.Slice(tt.want.Currencies, func(i, j int) bool {
				return tt.want.Currencies[i].Code < tt.want.Currencies[j].Code
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCommonCurrencies() = %v, want %v", got, tt.want)
			}
		})
	}
}
