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

// intraExchangePairWorker processes a slice of markets for a single exchange
// and creates the corresponding trading pairs, applying the taker fee.
func intraExchangePairWorker(markets []exchangeMarket, assetsPtr *models.Assets) models.Pairs {
	assets := *assetsPtr
	pairs := make(models.Pairs)

	// NOTE: even if there are duplicate markets with opposite sides (e.g. A/B, B/A)
	// they reflect each other anyways, so the last occurred one is applied to both sides

	for _, m := range markets {
		market := m.market
		exchangeId := m.exchangeId

		// retrieve asset information
		baseAssetKey := models.AssetKey{Exchange: exchangeId, Currency: market.Base}
		quoteAssetKey := models.AssetKey{Exchange: exchangeId, Currency: market.Quote}

		baseAsset, baseOk := assets[baseAssetKey]
		quoteAsset, quoteOk := assets[quoteAssetKey]

		// check if both assets exist
		if baseOk && quoteOk {
			// calculate fee multiplier
			feeMultiplier, _ := decimal.One.Quo(market.TakerFee)

			// create the "sell" pair (selling Base for Quote)
			// you sell the base asset at the bid price to receive the quote asset
			// the fee is taken from the quote asset you receive
			if market.Bid.Sign() > 0 { // ensure there is a valid bid price
				// Bid * (1 - TakerFee)
				effectiveRate, _ := market.Bid.Mul(feeMultiplier)

				pairKey := models.PairKey{From: baseAssetKey, To: quoteAssetKey}
				pairs[pairKey] = models.Pair{
					IntraExchange: true,
					Symbol:        market.Id,
					From:          baseAsset,
					To:            quoteAsset,
					Weight:        effectiveRate,
					Side:          "sell",
				}
			}

			// create the "buy" pair (buying Base with Quote)
			// you buy the base asset at the ask price using the quote asset
			// the fee is taken from the base asset you receive
			if market.Ask.Sign() > 0 { // Ensure there is a valid ask price.
				// the amount of Base you get for 1 Quote before fees is 1 / Ask
				// (1 / Ask)
				rate, _ := decimal.One.Quo(market.Ask)
				// (1 / Ask) * (1 - TakerFee)
				effectiveRate, _ := rate.Mul(feeMultiplier)

				pairKey := models.PairKey{From: quoteAssetKey, To: baseAssetKey}
				pairs[pairKey] = models.Pair{
					IntraExchange: true,
					Symbol:        market.Id,
					From:          quoteAsset,
					To:            baseAsset,
					Weight:        effectiveRate,
					Side:          "buy",
				}
			}
		}
	}
	return pairs
}
