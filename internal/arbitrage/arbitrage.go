package arbitrage

import (
	"log/slog"
	"maps"
	"runtime"
	"sync"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

// exchangeMarket is a helper struct to pass both exchangeId
// and market struct to the worker function
type exchangeMarket struct {
	exchangeId string
	market     models.Market
}

// createIntraExchangePairs creates trading pairs within each exchange.
// It calculates the total number of markets and distributes them across
// multiple concurrent workers to process them in parallel.
func createIntraExchangePairs(exchangesPtr *models.Exchanges, assetsPtr *models.Assets) models.Pairs {
	slog.Debug("creating intra-exchange pairs...")
	exchanges := *exchangesPtr
	allMarkets := make([]exchangeMarket, 0)

	// flatten the nested map of markets into a single slice
	for exchangeId, exchange := range exchanges {
		for _, market := range exchange.Markets {
			allMarkets = append(allMarkets, exchangeMarket{
				exchangeId: exchangeId,
				market:     market,
			})
		}
	}

	totalMarkets := len(allMarkets)
	if totalMarkets == 0 {
		return make(models.Pairs)
	}

	// determine the number of workers needed based on CPU core count
	numWorkers := min(runtime.GOMAXPROCS(0), totalMarkets)

	// calculate the chunk size for each worker
	marketsPerWorker := (totalMarkets + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	resultsChan := make(chan models.Pairs, numWorkers)

	// distribute and launch workers
	for i := range numWorkers {
		// calculate the slice of markets for the current worker
		start := i * marketsPerWorker
		end := start + marketsPerWorker
		end = min(end, totalMarkets)
		if start >= end {
			break // stop creating workers if no more work is available
		}

		workerMarkets := allMarkets[start:end]

		wg.Add(1)
		go func(markets []exchangeMarket) {
			defer wg.Done()
			workerResult := intraExchangePairWorker(markets, assetsPtr)
			resultsChan <- workerResult
		}(workerMarkets)
	}

	wg.Wait()
	close(resultsChan)

	// merge results into a single map
	finalPairs := make(models.Pairs)
	for workerResult := range resultsChan {
		maps.Copy(finalPairs, workerResult)
	}

	return finalPairs
}

// interExchangeCurrency represents a currency and exchanges which support it
type interExchangeCurrency struct {
	currency  string
	exchanges []string
}

// createInterExchangePairs creates trading pairs across exchanges.
// It calculates the total number of currencies and distributes them across
// multiple concurrent workers to process them in parallel.
func createInterExchangePairs(exchangesPtr *models.Exchanges, assetsPtr *models.Assets, capital decimal.Decimal) models.Pairs {
	slog.Debug("creating inter-exchange pairs...")
	exchanges := *exchangesPtr

	// create a map to store all unique currencies and the exchanges they are available on
	currencies := make(map[string][]string)
	for exchangeId, exchange := range exchanges {
		for currencyId := range exchange.Currencies {
			currencies[currencyId] = append(currencies[currencyId], exchangeId)
		}
	}

	// flatten the map into a slice of interExchangeCurrency structs
	allCurrencies := make([]interExchangeCurrency, 0, len(currencies))
	for currency, exchangeList := range currencies {
		allCurrencies = append(allCurrencies, interExchangeCurrency{
			currency:  currency,
			exchanges: exchangeList,
		})
	}

	totalCurrencies := len(allCurrencies)
	if totalCurrencies == 0 {
		return make(models.Pairs)
	}

	// determine the number of workers needed based on CPU core count
	numWorkers := min(runtime.GOMAXPROCS(0), totalCurrencies)

	// calculate the chunk size for each worker
	currenciesPerWorker := (totalCurrencies + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup
	resultsChan := make(chan models.Pairs, numWorkers)

	// distribute and launch workers
	for i := range numWorkers {
		// calculate the slice of currencies for the current worker
		start := i * currenciesPerWorker
		end := start + currenciesPerWorker
		end = min(end, totalCurrencies)
		if start >= end {
			break // stop creating workers if no more work is available
		}

		workerCurrencies := allCurrencies[start:end]

		wg.Add(1)
		go func(currencies []interExchangeCurrency) {
			defer wg.Done()
			workerResult := interExchangePairWorker(currencies, exchangesPtr, assetsPtr, capital)
			resultsChan <- workerResult
		}(workerCurrencies)
	}

	wg.Wait()
	close(resultsChan)

	// merge results into a single map
	finalPairs := make(models.Pairs)
	for workerResult := range resultsChan {
		maps.Copy(finalPairs, workerResult)
	}

	return finalPairs
}

func CreateAssetPairs(exchangesPtr *models.Exchanges, capital decimal.Decimal) (models.Assets, models.Index, models.Pairs) {
	if exchangesPtr == nil || len(*exchangesPtr) == 0 {
		slog.Warn("exchanges data is empty")
		return nil, nil, nil
	}
	slog.Info("creating asset pairs...")
	assets, index := createAssetIndex(exchangesPtr)

	pairs := make(models.Pairs)

	intraExchangePairs := createIntraExchangePairs(exchangesPtr, &assets)
	maps.Copy(pairs, intraExchangePairs)

	interExchangePairs := createInterExchangePairs(exchangesPtr, &assets, capital)
	maps.Copy(pairs, interExchangePairs)

	return assets, index, pairs
}
