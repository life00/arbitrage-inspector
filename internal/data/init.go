package data

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func findMissingItems(itemsToCheck []string, sourceList []string) []string {
	// create a map for efficient lookups
	sourceMap := make(map[string]struct{}, len(sourceList))
	for _, item := range sourceList {
		sourceMap[strings.ToLower(item)] = struct{}{}
	}

	var notFound []string
	for _, item := range itemsToCheck {
		if _, found := sourceMap[strings.ToLower(item)]; !found {
			notFound = append(notFound, item)
		}
	}
	return notFound
}

func validateExchanges(exchanges []string) error {
	if len(exchanges) == 0 {
		err := fmt.Errorf("list of exchanges is empty")
		slog.Error(err.Error())
		return err
	}
	supportedExchanges := []string{}

	for _, exchangeID := range ccxt.Exchanges {
		exchange := ccxt.CreateExchange(exchangeID, nil)

		// use the GetHas() method to check
		has := exchange.GetHas()
		if has["fetchCurrencies"] == true &&
			has["fetchMarkets"] == true &&
			has["fetchTickers"] == true &&
			has["createOrder"] == true &&
			has["fetchBalance"] == true &&
			has["withdraw"] == true &&
			has["fetchDepositAddress"] == true {
			supportedExchanges = append(supportedExchanges, exchangeID)
		}
	}

	// uses a reusable function
	invalidExchanges := findMissingItems(exchanges, supportedExchanges)

	if len(invalidExchanges) > 0 {
		err := fmt.Errorf("invalid exchanges: %s", strings.Join(invalidExchanges, ", "))
		slog.Error(err.Error())
		return err
	}

	return nil
}

// helper struct for loadCcxt()
type clientResult struct {
	client ccxt.IExchange
	err    error
}

// concurrently loads all exchanges with API credentials and fetches currency data into cache
func loadClient(exchanges []string) (models.Clients, error) {
	var wg sync.WaitGroup
	resultsChan := make(chan clientResult, len(exchanges))

	// concurrently load all exchanges
	for _, exchange := range exchanges {
		wg.Add(1)
		go func(ex string) {
			defer wg.Done()
			result := clientResult{}

			slog.Debug(fmt.Sprintf("loading exchange %s...", ex))

			// handle credentials from .env
			options := map[string]interface{}{}
			apiKeyEnvName := strings.ToUpper(ex) + "_API_KEY"
			secretEnvName := strings.ToUpper(ex) + "_SECRET"
			passwordEnvName := strings.ToUpper(ex) + "_PASSWORD"

			if apiKey := os.Getenv(apiKeyEnvName); apiKey != "" {
				options["apiKey"] = apiKey
			}
			if secret := os.Getenv(secretEnvName); secret != "" {
				options["secret"] = secret
			}
			if password := os.Getenv(passwordEnvName); password != "" {
				options["password"] = password
			}

			// instantiate the exchange object
			client := ccxt.CreateExchange(ex, options)

			if client == nil {
				result.err = fmt.Errorf("failed to create CCXT exchange for %s: exchange instance is nil", ex)
				resultsChan <- result
				return
			}
			result.client = client

			// load markets to cache data and test connection
			if _, err := client.LoadMarkets(); err != nil {
				result.err = fmt.Errorf("failed to load markets for %s: %w", ex, err)
				resultsChan <- result
				return
			}

			// fetch balance to test credentials
			if _, err := client.FetchBalance(); err != nil {
				result.err = fmt.Errorf("failed to authenticate for %s: %w", ex, err)
			}

			resultsChan <- result
		}(exchange)
	}

	wg.Wait()
	close(resultsChan)

	// extract results
	loadedClients := make(models.Clients)
	var allErrors []error
	for res := range resultsChan {
		if res.err != nil {
			allErrors = append(allErrors, res.err)
		} else {
			loadedClients[res.client.GetId()] = res.client
		}
	}

	// return errors if they occurred
	if len(allErrors) > 0 {
		var errorMessages []string
		for _, err := range allErrors {
			errorMessages = append(errorMessages, err.Error())
		}
		return nil, fmt.Errorf("errors occurred while loading exchanges: %s", strings.Join(errorMessages, "; "))
	}

	return loadedClients, nil
}

func validateCurrencies(currencies []string, clientsPtr *models.Clients) error {
	if len(currencies) == 0 {
		err := fmt.Errorf("list of currencies is empty")
		slog.Error(err.Error())
		return err
	}
	if clientsPtr == nil || len(*clientsPtr) == 0 {
		err := fmt.Errorf("list of clients is empty")
		slog.Error(err.Error())
		return err
	}

	clients := *clientsPtr

	var validCurrencies []string

	for _, e := range clients {
		for _, c := range e.GetCurrenciesList() {
			if c.Active == nil || !*c.Active || c.Deposit == nil || !*c.Deposit || c.Withdraw == nil || !*c.Withdraw || c.Id == nil {
				continue
			}
			validCurrencies = append(validCurrencies, *c.Id)
		}
	}

	missingCurrencies := findMissingItems(currencies, validCurrencies)

	if len(missingCurrencies) > 0 {
		err := fmt.Errorf("invalid currencies: %s", strings.Join(missingCurrencies, ", "))
		slog.Error(err.Error())
		return err
	}

	// no missing currencies
	return nil
}

// createExchange handles the logic for a single CCXT exchange client.
func createExchange(
	clientPtr *ccxt.IExchange,
	currencySet map[string]struct{},
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	exchanges models.Exchanges,
) {
	defer wg.Done()
	client := *clientPtr

	exchange := models.Exchange{
		Id:         client.GetId(),
		Markets:    []models.Market{},
		Currencies: make(map[string]models.Currency),
	}

	exchange.Markets = createMarkets(clientPtr, currencySet)
	exchange.Currencies = createCurrencies(clientPtr, currencySet)

	mu.Lock()
	exchanges[exchange.Id] = exchange
	mu.Unlock()
}

// createMarkets gets initial data and creates a Markets object.
func createMarkets(clientPtr *ccxt.IExchange, currencySet map[string]struct{}) []models.Market {
	client := *clientPtr

	var markets []models.Market
	marketsList := client.GetMarketsList()

	for _, m := range marketsList {
		// check all market conditions
		if m.Active != nil && *m.Active &&
			m.Spot != nil && *m.Spot &&
			m.Symbol != nil && m.BaseId != nil && m.QuoteId != nil {

			baseId := strings.ToUpper(*m.BaseId)
			quoteId := strings.ToUpper(*m.QuoteId)

			// check if both base and quote currencies are in the input set
			if _, baseExists := currencySet[baseId]; baseExists {
				if _, quoteExists := currencySet[quoteId]; quoteExists {
					markets = append(markets, models.Market{
						Id:    *m.Symbol,
						Base:  baseId,
						Quote: quoteId,
					})
				}
			}
		}
	}
	return markets
}

// createCurrencies gets initial data and creates a Currencies object.
func createCurrencies(clientPtr *ccxt.IExchange, currencySet map[string]struct{}) map[string]models.Currency {
	client := *clientPtr

	currenciesMap := make(map[string]models.Currency)

	// all currencies to the map
	for cur := range currencySet {
		currenciesMap[cur] = models.Currency{Id: cur}
	}

	// iterate through the clients currency list and add any that meet the criteria
	currenciesList := client.GetCurrenciesList()
	for _, cur := range currenciesList {
		// check all currency conditions
		if cur.Active != nil && *cur.Active &&
			cur.Deposit != nil && *cur.Deposit &&
			cur.Withdraw != nil && *cur.Withdraw &&
			cur.Id != nil {

			currencyId := *cur.Id
			if _, exists := currencySet[currencyId]; exists {
				currenciesMap[currencyId] = models.Currency{Id: currencyId}
			}
		}
	}

	return currenciesMap
}
