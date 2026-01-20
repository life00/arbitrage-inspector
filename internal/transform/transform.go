// Package transform provides functions to transform data into various structures.
package transform

import (
	"fmt"
	"log/slog"
	"maps"
	"runtime"
	"sync"

	"github.com/ccxt/ccxt/go/v4/pro"
	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

// CreateAssetIndex() creates asset index based on provided exchanges data structure
func CreateAssetIndex(exchangesPtr *models.Exchanges) (models.AssetIndexes, models.Index) {
	slog.Debug("creating asset index map")
	var i uint = 1 // 0 is for the super node
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
func CreateInterExchangePairs(config models.PairConfig, exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes, assetBalancesPtr *models.AssetBalances) models.Pairs {
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
			workerResult := interExchangePairWorker(config, currencies, exchangesPtr, assetsPtr, assetBalancesPtr)
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
func CreateIntraExchangePairs(config models.PairConfig, exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes) models.Pairs {
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
			workerResult := intraExchangePairWorker(config, markets, assetsPtr)
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

// CalculateEffectivePrices returns VWAP for asks and bids based on a target volume of Base currency.
func CalculateEffectivePrices(assetBalance models.AssetBalance, orderbook ccxtpro.OrderBook) (bid, ask decimal.Decimal) {
	balance := assetBalance.Balance
	if balance.IsZero() {
		return getTopOfOrderBook(orderbook)
	}

	ask = calculateOrderBookSide(balance, orderbook.Asks, "ask")
	bid = calculateOrderBookSide(balance, orderbook.Bids, "bid")

	// Fallback if calculated values are zero
	if ask.IsZero() || bid.IsZero() {
		tAsk, tBid := getTopOfOrderBook(orderbook)
		if ask.IsZero() {
			ask = tAsk
		}
		if bid.IsZero() {
			bid = tBid
		}
	}

	return bid, ask
}

func calculateOrderBookSide(targetVol decimal.Decimal, levels [][]float64, side string) decimal.Decimal {
	if len(levels) == 0 {
		return decimal.Zero
	}

	accumulatedVol := decimal.Zero
	totalQuoteValue := decimal.Zero

	for _, level := range levels {
		price, _ := decimal.NewFromFloat64(level[0])
		amount, _ := decimal.NewFromFloat64(level[1])

		remaining, _ := targetVol.Sub(accumulatedVol)

		// amount >= remaining
		if amount.Cmp(remaining) >= 0 {
			partialValue, _ := remaining.Mul(price)
			totalQuoteValue, _ = totalQuoteValue.Add(partialValue)
			accumulatedVol = targetVol
			break
		}

		fullLevelValue, _ := amount.Mul(price)
		totalQuoteValue, _ = totalQuoteValue.Add(fullLevelValue)
		accumulatedVol, _ = accumulatedVol.Add(amount)
	}

	// Warning if orderbook depth is less than target volume
	if accumulatedVol.Cmp(targetVol) < 0 {
		slog.Warn("insufficient liquidity",
			"side", side,
			"wanted", targetVol.String(),
			"found", accumulatedVol.String(),
		)
	}

	if accumulatedVol.IsZero() {
		return decimal.Zero
	}

	// average = totalValue / accumulatedVol
	avg, _ := totalQuoteValue.Quo(accumulatedVol)
	return avg
}

func getTopOfOrderBook(ob ccxtpro.OrderBook) (ask, bid decimal.Decimal) {
	if len(ob.Asks) > 0 {
		ask, _ = decimal.NewFromFloat64(ob.Asks[0][0])
	}
	if len(ob.Bids) > 0 {
		bid, _ = decimal.NewFromFloat64(ob.Bids[0][0])
	}
	return
}

// GetTransactionMarkets() retrieves all relevant markets from the ArbitragePath transactions
func GetTransactionMarkets(arbitragePath models.ArbitragePath, exchangesPtr *models.Exchanges) models.Exchanges {
	// extract all intra-transactions
	var intraTransactions models.TransactionPath
	for _, pair := range arbitragePath.Cycle {
		if pair.From.Exchange == pair.To.Exchange {
			intraTransactions = append(intraTransactions, pair)
		}
	}

	exchanges := *exchangesPtr
	localExchanges := make(models.Exchanges)

	for _, pair := range intraTransactions {
		exchange := pair.From.Exchange
		// get the market symbol
		var base string
		var quote string

		if _, ok := exchanges[exchange].Markets[fmt.Sprintf("%s/%s", pair.From.Currency, pair.To.Currency)]; ok {
			base = pair.From.Currency
			quote = pair.To.Currency
		} else {
			base = pair.To.Currency
			quote = pair.From.Currency
		}

		symbol := fmt.Sprintf("%s/%s", base, quote)

		// saving to the local exchanges
		localExchanges[exchange] = models.Exchange{
			ID: exchange,
			Currencies: map[string]models.Currency{
				pair.From.Currency: exchanges[exchange].Currencies[pair.From.Currency],
				pair.To.Currency:   exchanges[exchange].Currencies[pair.To.Currency],
			},
			Markets: map[string]models.Market{
				symbol: exchanges[exchange].Markets[symbol],
			},
		}
	}
	return localExchanges
}
