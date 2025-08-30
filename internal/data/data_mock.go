package data

import (
	"time"

	"github.com/ccxt/ccxt/go/v4"
)

// mockExchange implements the ccxt.IExchange interface for testing
type mockExchange struct {
	ccxt.IExchange
	name          string
	currencies    []ccxt.Currency
	apiCurrencies ccxt.Currencies
	markets       []ccxt.MarketInterface
	tickers       ccxt.Tickers
}

func (m *mockExchange) GetId() string {
	return m.name
}

func (m *mockExchange) GetCurrenciesList() []ccxt.Currency {
	return m.currencies
}

func (m *mockExchange) GetMarketsList() []ccxt.MarketInterface {
	return m.markets
}

func (m *mockExchange) FetchCurrencies(params ...interface{}) (ccxt.Currencies, error) {
	return m.apiCurrencies, nil
}

// type Currencies struct {
// 	Info       map[string]interface{}
// 	Currencies map[string]Currency
// }

// Add the missing FetchMarkets method to satisfy the ccxt.IExchange interface
func (m *mockExchange) FetchMarkets(params ...interface{}) ([]ccxt.MarketInterface, error) {
	return m.markets, nil
}

func (m *mockExchange) FetchTickers(options ...ccxt.FetchTickersOptions) (ccxt.Tickers, error) {
	return m.tickers, nil
}

// type Tickers struct {
// 	Info    map[string]interface{}
// 	Tickers map[string]Ticker
// }

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

func newMarket(symbol, baseId, quoteId string, active, spot bool) ccxt.MarketInterface {
	return ccxt.MarketInterface{
		Symbol:  &symbol,
		BaseId:  &baseId,
		QuoteId: &quoteId,
		Active:  &active,
		Spot:    &spot,
	}
}

func newTicker(symbol string, bid, ask float64) ccxt.Ticker {
	time := time.Now().UnixNano()
	return ccxt.Ticker{
		Symbol:    &symbol,
		Bid:       &bid,
		Ask:       &ask,
		Timestamp: &time,
	}
}
