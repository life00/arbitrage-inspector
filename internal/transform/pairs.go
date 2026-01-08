package transform

import (
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
func intraExchangePairWorker(config models.PairConfig, markets []exchangeMarket, assetsPtr *models.AssetIndexes) models.Pairs {
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
			// determine the fee multiplier based on the configuration
			var feeMultiplier decimal.Decimal

			switch config.IntraType {
			case models.FeeTypeNominal:
				feeMultiplier = decimal.MustNew(1, 0)
			case models.FeeTypeEffective:
				// calculate fee multiplier
				if market.TakerFee.Sign() == -1 {
					slog.Warn("taker fee is negative", "exchange", exchangeId, "market", market.Id)
					continue
				}
				// "sell" pair
				// Bid * (1 - TakerFee)
				// "buy" pair
				// (1 / Ask) * (1 - TakerFee)
				feeMultiplier, _ = decimal.One.Sub(market.TakerFee)
				if feeMultiplier.Sign() != 1 {
					slog.Warn("taker fee >= 100%; skipping", "exchange", exchangeId, "market", market.Id)
					continue
				}
			case models.FeeTypeConstant:
				// this is not a possible option
				// assuming nominal behavior
				feeMultiplier = decimal.MustNew(1, 0)
			}

			// create the "sell" pair (selling Base for Quote)
			// you sell the base asset at the bid price to receive the quote asset
			// the fee is taken from the quote asset you receive
			if market.Bid.Sign() > 0 { // ensure there is a valid bid price
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
	config models.PairConfig,
	currencies []interExchangeCurrency,
	exchangesPtr *models.Exchanges,
	assetsPtr *models.AssetIndexes,
	assetBalancesPtr *models.AssetBalances,
) models.Pairs {
	exchanges := *exchangesPtr
	assets := *assetsPtr
	assetBalances := *assetBalancesPtr
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
					slog.Warn("missing exchange information", "from", fromExchangeId, "to", toExchangeId)
					continue
				}

				fromCurrency, fromCurrencyOk := fromExchange.Currencies[c.currency]
				toCurrency, toCurrencyOk := toExchange.Currencies[c.currency]
				if !fromCurrencyOk || !toCurrencyOk {
					slog.Warn("missing currency information", "currency", c.currency, "from", fromExchangeId, "to", toExchangeId)
					continue
				}

				// find common networks and the one with the cheapest withdrawal fee
				var cheapestNetwork string
				actualMinFee := decimal.Zero
				firstCommonNetworkFound := false

				for fromNetworkId, fromNetwork := range fromCurrency.Networks {
					if _, ok := toCurrency.Networks[fromNetworkId]; ok {
						// use Cmp to compare decimal values; Cmp returns -1 if less than, 0 if equal, 1 if greater than
						if !firstCommonNetworkFound || fromNetwork.WithdrawalFee.Cmp(actualMinFee) < 0 {
							actualMinFee = fromNetwork.WithdrawalFee
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
					slog.Warn("missing asset information", "currency", c.currency, "from", fromExchangeId, "to", toExchangeId)
					continue
				}

				// determine fee to apply based on configuration
				feeAmount := decimal.Zero

				switch config.InterType {
				case models.FeeTypeNominal:
					feeAmount = decimal.Zero
				case models.FeeTypeEffective:
					// capital balance in the local currency
					localCapital := assetBalances[fromAssetKey].Balance

					// calculate effective rate after network fees
					// effectiveRate = (LocalCapital - Fee) / LocalCapital
					effectiveLocalCapital, _ := localCapital.Sub(actualMinFee)

					if effectiveLocalCapital.Sign() <= 0 {
						// only warn for Effective mode
						// in Constant mode, this might be expected for small test capitals
						slog.Warn("withdrawal fee higher than capital; skipping", "exchange", fromExchangeId, "currency", fromCurrency.Id)
						continue
					}

					effectiveRate, _ := effectiveLocalCapital.Quo(localCapital)

					pairKey := models.PairKey{From: fromAssetKey, To: toAssetKey}
					pairs[pairKey] = models.Pair{
						IntraExchange: false,
						From:          fromAsset,
						To:            toAsset,
						Weight:        effectiveRate,
						Network:       cheapestNetwork,
					}
					// the process is over, so continue to next pair
					continue
				case models.FeeTypeConstant:
					feeAmount = config.ConstantFee
				}

				// validate fee and capital
				if feeAmount.Sign() < 0 {
					continue
				}
				if config.Capital.Sign() <= 0 {
					continue
				}

				// calculate effective rate after network fees
				// effectiveRate = (Capital - Fee) / Capital
				effectiveCapital, _ := config.Capital.Sub(feeAmount)
				if effectiveCapital.Sign() <= 0 {
					// Only warn for Effective mode; in Constant mode, this might be expected for small test capitals
					if config.InterType == models.FeeTypeEffective {
						slog.Warn("withdrawal fee higher than capital; skipping", "exchange", fromExchangeId, "currency", fromCurrency.Id)
					}
					continue
				}
				effectiveRate, _ := effectiveCapital.Quo(config.Capital)

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
