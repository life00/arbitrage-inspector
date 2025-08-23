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

func validateExchanges(exchanges models.Exchanges) error {
	if len(exchanges.Exchanges) == 0 {
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

	// extract the names from the Exchanges object
	exchangeNames := make([]string, len(exchanges.Exchanges))
	for i, exchange := range exchanges.Exchanges {
		exchangeNames[i] = exchange.Name
	}
	// uses a reusable function
	invalidExchanges := findMissingItems(exchangeNames, supportedExchanges)

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

func getCommonValidCurrencies(ccxtExchangesPtr *[]ccxt.IExchange) models.Currencies {
	if ccxtExchangesPtr == nil || len(*ccxtExchangesPtr) == 0 {
		return models.Currencies{}
	}

	ccxtExchanges := *ccxtExchangesPtr

	firstExchangeCurrencies := make(map[string]bool)
	for _, currency := range ccxtExchanges[0].GetCurrenciesList() {
		// it also checks if the currency is active or not
		// it doesn't accept any inactive currencies or those that do not accept deposits/withdrawals
		if currency.Active == nil || !*currency.Active || currency.Deposit == nil || !*currency.Deposit || currency.Withdraw == nil || !*currency.Withdraw || currency.Id == nil {
			continue
		}
		firstExchangeCurrencies[*currency.Id] = true
	}

	commonCurrencies := firstExchangeCurrencies

	for i := 1; i < len(ccxtExchanges); i++ {
		currentExchangeCurrencies := make(map[string]bool)
		for _, currency := range ccxtExchanges[i].GetCurrenciesList() {
			if currency.Active == nil || !*currency.Active || currency.Deposit == nil || !*currency.Deposit || currency.Withdraw == nil || !*currency.Withdraw || currency.Id == nil {
				continue
			}
			currentExchangeCurrencies[*currency.Id] = true
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
		result = append(result, models.Currency{Id: currency})
	}

	return models.Currencies{Currencies: result}
}

func validateCurrencies(currencies models.Currencies, commonCurrencies models.Currencies) error {
	if len(currencies.Currencies) == 0 {
		err := fmt.Errorf("list of currencies is empty")
		slog.Error(err.Error())
		return err
	}

	// extract currency ID into slices of strings
	var currencyIds []string
	for _, c := range currencies.Currencies {
		currencyIds = append(currencyIds, c.Id)
	}

	var commonCurrencyIds []string
	for _, c := range commonCurrencies.Currencies {
		commonCurrencyIds = append(commonCurrencyIds, c.Id)
	}

	missingCurrencies := findMissingItems(currencyIds, commonCurrencyIds)

	if len(missingCurrencies) > 0 {
		err := fmt.Errorf("invalid currencies: %s", strings.Join(missingCurrencies, ", "))
		slog.Error(err.Error())
		return err
	}

	// no missing currencies
	return nil
}

// NOTE: maybe it might be possible to have a reusable generic function
// for getCommonValidMarkets() and getCommonValidCurrencies()?
// also split the common market and valid market identification into two functions?
func getCommonValidMarkets(ccxtExchangesPtr *[]ccxt.IExchange) models.Markets {
	if ccxtExchangesPtr == nil || len(*ccxtExchangesPtr) == 0 {
		return models.Markets{}
	}

	ccxtExchanges := *ccxtExchangesPtr

	firstExchangeMarkets := make(map[string]models.Market)
	for _, market := range ccxtExchanges[0].GetMarketsList() {
		// verify if the market is supported
		if market.Active == nil || !*market.Active || market.Spot == nil || !*market.Spot || market.Symbol == nil || market.BaseId == nil || market.QuoteId == nil {
			continue
		}

		firstExchangeMarkets[*market.Symbol] = models.Market{
			Id:    *market.Symbol,
			Base:  *market.BaseId,
			Quote: *market.QuoteId,
		}
	}

	commonMarkets := firstExchangeMarkets

	for i := 1; i < len(ccxtExchanges); i++ {
		currentExchangeMarkets := make(map[string]bool)
		for _, market := range ccxtExchanges[i].GetMarketsList() {
			if market.Active == nil || !*market.Active || market.Spot == nil || !*market.Spot || market.Symbol == nil || market.BaseId == nil || market.QuoteId == nil {
				continue
			}
			currentExchangeMarkets[*market.Symbol] = true
		}

		// finds the intersection
		for marketId := range commonMarkets {
			if !currentExchangeMarkets[marketId] {
				delete(commonMarkets, marketId)
			}
		}
	}

	// convert the result into the output
	result := make([]models.Market, 0, len(commonMarkets))
	for _, market := range commonMarkets {
		result = append(result, market)
	}

	return models.Markets{Markets: result}
}
