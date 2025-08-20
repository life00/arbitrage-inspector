package data

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/ccxt/ccxt/go/v4"
	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func containsAll(itemsToCheck []string, sourceList []string) []string {
	// Create a map for efficient lookups.
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

func validateExchanges(exchanges models.Exchanges) error {
	// extract the names from the Exchanges object
	exchangeNames := make([]string, len(exchanges.Exchanges))
	for i, exchange := range exchanges.Exchanges {
		exchangeNames[i] = exchange.Name
	}

	// uses a reusable function
	invalidExchanges := containsAll(exchangeNames, ccxt.Exchanges)

	if len(invalidExchanges) > 0 {
		err := fmt.Errorf("invalid exchanges: %s", strings.Join(invalidExchanges, ", "))
		slog.Error(err.Error())
		return err
	}

	return nil
}

// helper struct for loadCcxt()
type exchangeResult struct {
	exchange ccxt.IExchange
	err      error
}

// concurrently loads all exchanges with API credentials and fetches currency data into cache
func loadCcxt(exchanges models.Exchanges) ([]ccxt.IExchange, error) {
	// get the environment variables
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env file")
	}

	var wg sync.WaitGroup
	resultsChan := make(chan exchangeResult, len(exchanges.Exchanges))

	// concurrently load all exchanges
	for _, exchange := range exchanges.Exchanges {
		wg.Add(1)
		go func(ex models.Exchange) {
			defer wg.Done()
			result := exchangeResult{}

			slog.Debug(fmt.Sprintf("loading exchange %s...", ex.Name))

			// handle credentials from .env
			options := map[string]interface{}{}
			apiKeyEnvName := strings.ToUpper(ex.Name) + "_API_KEY"
			secretEnvName := strings.ToUpper(ex.Name) + "_SECRET"
			passwordEnvName := strings.ToUpper(ex.Name) + "_PASSWORD"

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
			ccxtExchange := ccxt.CreateExchange(ex.Name, options)

			if ccxtExchange == nil {
				result.err = fmt.Errorf("failed to create CCXT exchange for %s: exchange instance is nil", ex.Name)
				resultsChan <- result
				return
			}
			result.exchange = ccxtExchange

			// load markets to cache data and test connection
			if _, err := ccxtExchange.LoadMarkets(); err != nil {
				result.err = fmt.Errorf("failed to load markets for %s: %w", ex.Name, err)
			}

			resultsChan <- result
		}(exchange)
	}

	wg.Wait()
	close(resultsChan)

	// extract results
	var loadedExchanges []ccxt.IExchange
	var allErrors []error
	for res := range resultsChan {
		if res.err != nil {
			allErrors = append(allErrors, res.err)
		} else {
			loadedExchanges = append(loadedExchanges, res.exchange)
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

	return loadedExchanges, nil
}

func getCommonCurrencies(ccxtExchangesPtr *[]ccxt.IExchange) models.Currencies {
	ccxtExchanges := *ccxtExchangesPtr

	if len(ccxtExchanges) == 0 {
		return models.Currencies{}
	}

	firstExchangeCurrencies := make(map[string]bool)
	for _, currency := range ccxtExchanges[0].GetCurrenciesList() {
		if currency.Id != nil {
			firstExchangeCurrencies[*currency.Id] = true
		}
	}

	commonCurrencies := firstExchangeCurrencies

	for i := 1; i < len(ccxtExchanges); i++ {
		currentExchangeCurrencies := make(map[string]bool)
		for _, currency := range ccxtExchanges[i].GetCurrenciesList() {
			if currency.Id != nil {
				currentExchangeCurrencies[*currency.Id] = true
			}
		}

		// find the intersection
		for currency := range commonCurrencies {
			if !currentExchangeCurrencies[currency] {
				delete(commonCurrencies, currency)
			}
		}
	}

	// convert the result into models.Currencies
	result := make([]models.Currency, 0, len(commonCurrencies))
	for currency := range commonCurrencies {
		result = append(result, models.Currency{Code: currency})
	}

	return models.Currencies{Currencies: result}
}
