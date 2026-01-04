package transform

import (
	"fmt"
	"log/slog"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

// exchangeMarket is a helper struct to pass both exchangeId
// and market struct to the worker function
type exchangeMarket struct {
	exchangeId string
	market     models.Market
}

// intraExchangePairWorker processes a slice of markets for a single exchange
// and creates the corresponding trading pairs, applying the taker fee.
func intraExchangePairWorker(markets []exchangeMarket, assetsPtr *models.AssetIndexes) models.Pairs {
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
			feeMultiplier, _ := decimal.One.Sub(market.TakerFee)
			if market.TakerFee.Sign() == -1 {
				slog.Warn("taker fee is negative", "exchange", exchangeId, "market", market.Id)
				continue
			}
			if feeMultiplier.Sign() != 1 {
				slog.Warn("taker fee >= 100%; skipping", "exchange", exchangeId, "market", market.Id)
				continue
			}

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

// interExchangeCurrency represents a currency and exchanges which support it
type interExchangeCurrency struct {
	currency  string
	exchanges []string
}

// interExchangePairWorker processes a slice of currencies and creates the corresponding
// inter-exchange trading pairs.
func interExchangePairWorker(
	currencies []interExchangeCurrency,
	exchangesPtr *models.Exchanges,
	assetsPtr *models.AssetIndexes,
	capital decimal.Decimal,
) models.Pairs {
	exchanges := *exchangesPtr
	assets := *assetsPtr
	pairs := make(models.Pairs)

	for _, c := range currencies {
		// create pairs for all combinations of exchanges for the current currency
		for _, fromExchangeId := range c.exchanges {
			for _, toExchangeId := range c.exchanges {
				// avoid creating a pair to and from the same exchange
				if fromExchangeId == toExchangeId {
					continue
				}

				// retrieve exchange and currency information
				fromExchange, fromExchangeOk := exchanges[fromExchangeId]
				toExchange, toExchangeOk := exchanges[toExchangeId]
				if !fromExchangeOk || !toExchangeOk {
					slog.Warn("missing exchange information for %s or %s", fromExchangeId, toExchangeId)
					continue
				}

				fromCurrency, fromCurrencyOk := fromExchange.Currencies[c.currency]
				toCurrency, toCurrencyOk := toExchange.Currencies[c.currency]
				if !fromCurrencyOk || !toCurrencyOk {
					slog.Warn(fmt.Sprintf("missing currency information for %s on exchange %s or %s", c.currency, fromExchangeId, toExchangeId))
					continue
				}

				// find common networks and the one with the cheapest withdrawal fee
				var cheapestNetwork string
				minFee := decimal.Zero
				firstCommonNetworkFound := false

				for fromNetworkId, fromNetwork := range fromCurrency.Networks {
					if _, ok := toCurrency.Networks[fromNetworkId]; ok {
						// use Cmp to compare decimal values; Cmp returns -1 if less than, 0 if equal, 1 if greater than
						if !firstCommonNetworkFound || fromNetwork.WithdrawalFee.Cmp(minFee) < 0 {
							minFee = fromNetwork.WithdrawalFee
							cheapestNetwork = fromNetworkId
							firstCommonNetworkFound = true
						}
					}
				}

				if !firstCommonNetworkFound {
					continue
				}

				// create asset keys and retrieve asset information
				fromAssetKey := models.AssetKey{Exchange: fromExchangeId, Currency: c.currency}
				toAssetKey := models.AssetKey{Exchange: toExchangeId, Currency: c.currency}

				fromAsset, fromAssetOk := assets[fromAssetKey]
				toAsset, toAssetOk := assets[toAssetKey]
				if !fromAssetOk || !toAssetOk {
					slog.Warn("missing asset information for %s on %s or %s on %s", c.currency, fromExchangeId, c.currency, toExchangeId)
					continue
				}

				// calculate the effective rate after withdrawal fees
				if minFee.Sign() < 0 {
					continue
				}

				if capital.Sign() <= 0 {
					continue
				}

				effectiveCapital, _ := capital.Sub(minFee)
				if effectiveCapital.Sign() <= 0 {
					slog.Warn("withdrawal fee higher than capital; skipping", "exchange", fromExchangeId, "currency", fromCurrency.Id)
					continue
				}
				effectiveRate, _ := effectiveCapital.Quo(capital)

				pairKey := models.PairKey{From: fromAssetKey, To: toAssetKey}
				pairs[pairKey] = models.Pair{
					IntraExchange: false,
					From:          fromAsset,
					To:            toAsset,
					Weight:        effectiveRate,
					Network:       cheapestNetwork,
				}
			}
		}
	}

	return pairs
}
