package fetch

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ccxt/ccxt/go/v4/pro"
	"github.com/joho/godotenv"
)

// TestExchange implements the ccxtpro.IExchange interface for testing
type TestExchange struct {
	ccxtpro.IExchange
	Name                 string
	Currencies           []ccxtpro.Currency
	APICurrencies        ccxtpro.Currencies
	Markets              []ccxtpro.MarketInterface
	Tickers              ccxtpro.Tickers
	FetchTickersError    error
	FetchCurrenciesError error
	FetchMarketsError    error
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})))
	err := godotenv.Load("../../.env")
	if err != nil {
		slog.Error("failed to load .env file")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func (m *TestExchange) GetId() string {
	return m.Name
}

func (m *TestExchange) GetCurrenciesList() []ccxtpro.Currency {
	return m.Currencies
}

func (m *TestExchange) GetMarketsList() []ccxtpro.MarketInterface {
	return m.Markets
}

func (m *TestExchange) FetchCurrencies(args ...interface{}) (ccxtpro.Currencies, error) {
	if m.FetchCurrenciesError != nil {
		return ccxtpro.Currencies{}, m.FetchCurrenciesError
	}
	return m.APICurrencies, nil
}

func (m *TestExchange) FetchMarkets(args ...interface{}) ([]ccxtpro.MarketInterface, error) {
	if m.FetchMarketsError != nil {
		return nil, m.FetchMarketsError
	}
	return m.Markets, nil
}

func (m *TestExchange) FetchTickers(options ...ccxtpro.FetchTickersOptions) (ccxtpro.Tickers, error) {
	if m.FetchTickersError != nil {
		return ccxtpro.Tickers{}, m.FetchTickersError
	}
	return m.Tickers, nil
}

func newMockCurrency(code string) ccxtpro.Currency {
	active := true
	deposit := true
	withdraw := true
	return ccxtpro.Currency{
		Code:     &code,
		Active:   &active,
		Deposit:  &deposit,
		Withdraw: &withdraw,
	}
}

func newMockMarket(symbol, baseId, quoteId string, active, spot bool) ccxtpro.MarketInterface {
	return ccxtpro.MarketInterface{
		Symbol:  &symbol,
		BaseId:  &baseId,
		QuoteId: &quoteId,
		Active:  &active,
		Spot:    &spot,
	}
}

func newMockTicker(symbol string, bid, ask float64) ccxtpro.Ticker {
	timestamp := time.Now().UnixNano()
	return ccxtpro.Ticker{
		Symbol:    &symbol,
		Bid:       &bid,
		Ask:       &ask,
		Timestamp: &timestamp,
	}
}

func newString(s string) *string {
	return &s
}

func newBool(b bool) *bool {
	return &b
}

func newFloat64(f float64) *float64 {
	return &f
}

func newInt64(i int64) *int64 {
	return &i
}

func newMockError(s string) error {
	return &mockError{s: s}
}

type mockError struct {
	s string
}

func (e *mockError) Error() string {
	return e.s
}
