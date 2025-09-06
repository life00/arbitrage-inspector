package arbitrage

import (
	"log/slog"

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
