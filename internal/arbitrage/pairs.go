package arbitrage

import (
	"log/slog"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func createAssetIndex(exchangesPtr *models.Exchanges) (models.Assets, models.Index) {
	slog.Debug("creating asset index map...")
	var i uint
	assets := make(models.Assets)
	index := make(models.Index)
	// looping through all possible currencies in all exchanges
	for exchangeId, exchange := range *exchangesPtr {
		for currencyId := range exchange.Currencies {
			// creating asset map
			assets[models.AssetKey{
				Exchange: exchangeId,
				Currency: currencyId,
			}] = models.Asset{
				Exchange: exchangeId,
				Currency: currencyId,
				Index:    i,
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

func intraExchangePairWorker(markets []exchangeMarket, assetsPtr *models.Assets) models.Pairs {
	assets := *assetsPtr
	pairs := make(models.Pairs)

	for _, m := range markets {
		market := m.market
		exchangeId := m.exchangeId

		// define asset keys
		fromAssetKey := models.AssetKey{Exchange: exchangeId, Currency: market.Base}
		toAssetKey := models.AssetKey{Exchange: exchangeId, Currency: market.Quote}

		// retrieve full Asset objects
		fromAsset, fromOk := assets[fromAssetKey]
		toAsset, toOk := assets[toAssetKey]
		// TODO: account for taker/maker fees
		// FIXME: weight seems to be calculated incorrectly
		// FIXME: sometimes the markets might have both A/B and B/A sides available
		// they should prioritized and handled appropriately

		// only process if both assets exist
		if fromOk && toOk {
			// create the "sell" pair (selling base for quote)
			// the weight is the bid price (what someone is willing to pay for the base)
			if market.Bid.Sign() > 0 { // ensure there is a valid bid price
				sellPairKey := models.PairKey{From: fromAssetKey, To: toAssetKey}
				pairs[sellPairKey] = models.Pair{
					Symbol: market.Id,
					From:   fromAsset,
					To:     toAsset,
					Weight: market.Bid,
					Side:   "sell",
				}
			}

			// Create the "Buy" pair (buying Base with Quote)
			// The weight is the ask price (what someone is asking for the base)
			if market.Ask.Sign() > 0 { // Ensure there is a valid ask price
				weight, _ := decimal.MustNew(1, 0).Quo(market.Ask)
				buyPairKey := models.PairKey{From: toAssetKey, To: fromAssetKey}
				pairs[buyPairKey] = models.Pair{
					Symbol: market.Id,
					From:   toAsset,
					To:     fromAsset,
					Weight: weight, // The weight is 1 / ask_price
					Side:   "buy",
				}
			}
		}
	}
	return pairs
}
