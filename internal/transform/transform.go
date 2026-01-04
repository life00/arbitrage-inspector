package transform

import (
	"log/slog"
	"maps"
	"runtime"
	"sync"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

// CreateAssetIndex() creates asset index based on provided exchanges data structure
func CreateAssetIndex(exchangesPtr *models.Exchanges) (models.AssetIndexes, models.Index) {
	slog.Debug("creating asset index map...")
	var i uint
	assets := make(models.AssetIndexes)
	index := make(models.Index)
	// looping through all possible currencies in all exchanges
	for exchangeId, exchange := range *exchangesPtr {
		for currencyId := range exchange.Currencies {
			// creating asset map
			assets[models.AssetKey{
				Exchange: exchangeId,
				Currency: currencyId,
			}] = models.AssetIndex{
				Asset: models.AssetKey{
					Exchange: exchangeId,
					Currency: currencyId,
				},
				Index: i,
			}
			// creating index map
			index[i] = models.AssetKey{
				Exchange: exchangeId,
				Currency: currencyId,
			}
			i++
		}
	}
	return assets, index
}

// CreateInterExchangePairs creates trading pairs across exchanges.
// It calculates the total number of currencies and distributes them across
// multiple concurrent workers to process them in parallel.
func CreateInterExchangePairs(exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes, capital decimal.Decimal) models.Pairs {
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

// CreateIntraExchangePairs creates trading pairs within each exchange.
// It calculates the total number of markets and distributes them across
// multiple concurrent workers to process them in parallel.
func CreateIntraExchangePairs(exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes) models.Pairs {
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
