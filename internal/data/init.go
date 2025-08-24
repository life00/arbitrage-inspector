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
			has["fetchDepositWithdrawFees"] == true &&
			// has["fetchTradingFees"] == true && // NOTE: I don't think this feature is necessary; this can be found in fetchCurrencies()
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

// func getCommonValidMarkets(ccxtExchangesPtr *[]ccxt.IExchange) models.Markets {
// 	if ccxtExchangesPtr == nil || len(*ccxtExchangesPtr) == 0 {
// 		return models.Markets{}
// 	}
//
// 	// Define the function to process a single market.
// 	processMarket := func(market ccxt.MarketInterface) (string, models.Market, bool) {
// 		// Validation logic for a market.
// 		if market.Active == nil || !*market.Active || market.Spot == nil || !*market.Spot || market.Symbol == nil || market.BaseId == nil || market.QuoteId == nil {
// 			return "", models.Market{}, false
// 		}
// 		model := models.Market{
// 			Id:    *market.Symbol,
// 			Base:  *market.BaseId,
// 			Quote: *market.QuoteId,
// 		}
// 		return model.Id, model, true
// 	}
//
// 	// Call the generic helper.
// 	commonMarketsMap := getCommonItems(
// 		*ccxtExchangesPtr,
// 		func(e ccxt.IExchange) []ccxt.MarketInterface { return e.GetMarketsList() },
// 		processMarket,
// 	)
//
// 	// Convert the result map into the final slice.
// 	result := make([]models.Market, 0, len(commonMarketsMap))
// 	for _, market := range commonMarketsMap {
// 		result = append(result, market)
// 	}
//
// 	return models.Markets{Markets: result}
// }

// getMatchingMarkets finds all markets where both the base and quote currencies
// are present in the provided list of currencies
// func getMatchingMarkets(commonMarkets models.Markets, currencies models.Currencies) models.Markets {
// 	// set for quick lookups
// 	currencySet := make(map[string]struct{})
// 	for _, currency := range currencies.Currencies {
// 		currencySet[currency.Id] = struct{}{}
// 	}
//
// 	var matchingMarkets []models.Market
//
// 	for _, market := range commonMarkets.Markets {
// 		_, hasBase := currencySet[market.Base]
// 		_, hasQuote := currencySet[market.Quote]
//
// 		if hasBase && hasQuote {
// 			matchingMarkets = append(matchingMarkets, market)
// 		}
// 	}
//
// 	return models.Markets{Markets: matchingMarkets}
// }
