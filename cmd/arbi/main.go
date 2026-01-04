package main

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"time"

	"github.com/govalues/decimal"
	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/engine"
	"github.com/life00/arbitrage-inspector/internal/fetch"
	"github.com/life00/arbitrage-inspector/internal/models"
	"github.com/life00/arbitrage-inspector/internal/trade"
	"github.com/life00/arbitrage-inspector/internal/transform"
	"github.com/lmittmann/tint"
)

// initialization step
func initialization() (models.Config, models.Exchanges, models.Clients, models.AssetIndexes, models.Index, error) {
	// setup a default logger
	logger := slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			// AddSource: true,
			Level: slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)

	// get the environment variables (API credentials)
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env file")
		return models.Config{}, nil, nil, nil, nil, err
	}

	// TODO: Parse cli arguments, config file, etc.
	// and define the config structure

	config := models.Config{
		Exchanges: []string{
			"binance",
			"kucoin",
			"bitget",
			// "htx",
			// "coinbase",
		},
		CurrencyInputMode: models.AllCurrencies,
		Currencies: []string{
			"BTC",
			"ETH",
			"USDC",
			"DOGE",
			"SOL",
			"BNB",
			"USDT",
			"BCH",
			"LTC",
			"XMR",
		},
		ExcludedCurrencies: []string{
			// problematic currency codes
			"NEIRO",
			"BROCCOLI",
		},
	}

	// initialization of exchanges data structure
	exchanges, clients, err := fetch.InitializeExchanges(config)
	if err != nil {
		return models.Config{}, nil, nil, nil, nil, err
	}

	// creation of asset index
	assets, index := transform.CreateAssetIndex(&exchanges)

	return config, exchanges, clients, assets, index, err
}

// periodic update step
func periodicUpdate(
	configPtr *models.Config,
	exchangesPtr *models.Exchanges,
	clientsPtr *models.Clients,
	assetsPtr *models.AssetIndexes,
	indexPtr *models.Index,
) (models.Pairs, error) {
	// update exchange data structure
	err := fetch.UpdateExchanges(exchangesPtr, clientsPtr, true, true)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// saveAnyJson(exchanges, "/home/user/dev/src/arbitrage/exchanges.json")
	// load data from cached exchanges.json
	// exchanges, err := loadAnyJson[models.Exchanges]("/home/user/dev/src/arbitrage/exchanges.json")
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// find balances of all assets
	_, err = findAssetBalances(configPtr, exchangesPtr, clientsPtr, assetsPtr, indexPtr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// create effective inter-exchange pairs
	interPairs, err := createInterPairs(configPtr, exchangesPtr, assetsPtr, indexPtr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return interPairs, nil
}

// finds source asset balances
func findAssetBalances(configPtr *models.Config, exchangesPtr *models.Exchanges, clientsPtr *models.Clients, assetsPtr *models.AssetIndexes, indexPtr *models.Index) (map[models.AssetKey]models.AssetBalance, error) {
	// TODO:
	// transform: create nominal intra-exchange pairs (no fees)
	// transform: create nominal inter-exchange pairs (no fees)
	// engine: find balances of all assets
	// transform: config.SourceAssets balances
	// trade: ensure sufficient balances

	// FIXME: temporary solution
	*configPtr = models.Config{
		ReferenceAsset: models.AssetBalance{
			Asset: models.AssetKey{
				Exchange: "binance",
				Currency: "USDC",
			},
			Balance: decimal.MustNew(1000000, 0),
		},
		SourceAssets: map[models.AssetKey]models.AssetBalance{
			{Exchange: "binance", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				Balance: decimal.MustNew(1000000, 0),
			},
			{Exchange: "kucoin", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "kucoin", Currency: "USDC"},
				Balance: decimal.MustNew(1000000, 0),
			},
			{Exchange: "bitget", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "bitget", Currency: "USDC"},
				Balance: decimal.MustNew(1000000, 0),
			},
		},
	}

	return nil, nil
}

// creates effective inter-exchange pairs
func createInterPairs(configPtr *models.Config, exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes, indexPtr *models.Index) (models.Pairs, error) {
	// TODO:
	// transform: create effective intra-exchange pairs (with regular bid/ask prices)
	// transform: create effective inter-exchange pairs (with some constant amount denominated in ReferenceAsset like 1 USDT)
	// engine: find balances of all assets
	// transform: create actual effective inter-exchange pairs (using all asset balances)

	// FIXME:temporary solution
	interPairs := transform.CreateInterExchangePairs(exchangesPtr, assetsPtr, configPtr.ReferenceAsset.Balance)

	return interPairs, nil
}

// continuous update step
func continuousUpdate(
	configPtr *models.Config,
	exchangesPtr *models.Exchanges,
	clientsPtr *models.Clients,
	assetsPtr *models.AssetIndexes,
	indexPtr *models.Index,
	pairsPtr *models.Pairs,
) (bool, models.ArbitragePath, error) {
	// TODO:
	// client: wait some time
	// watch: call watcher to update data
	// transform: create actual effective inter-exchange pairs
	// engine: search for reasonable arbitrage and find fill ArbitragePath
	// trade: arbitrage is profitable?

	// TODO: watcher
	// watch: initialize orderbook watcher (establish websocket connections)
	// watch: cache all received orderbook data
	// transform: calculate effective prices (from orderbook data)
	// watch: update exchanges data structure

	// FIXME: temporary solution

	// wait some time before each update
	time.Sleep(60 * time.Second)

	// update exchange price data using regular bid/ask prices
	err := fetch.UpdateExchanges(exchangesPtr, clientsPtr, false, false)
	if err != nil {
		return false, models.ArbitragePath{}, nil
	}

	// calculate regular intra-exchange pairs
	intraPairs := transform.CreateIntraExchangePairs(exchangesPtr, assetsPtr)
	maps.Copy(*pairsPtr, intraPairs)

	// type SerializedPair struct {
	// 	Key   models.PairKey
	// 	Value models.Pair
	// }
	// var serializedPairs []SerializedPair
	// for key, value := range pairs {
	// 	serializedPairs = append(serializedPairs, SerializedPair{
	// 		Key:   key,
	// 		Value: value,
	// 	})
	// }
	// saveAnyJson(serializedPairs, "/home/user/dev/src/arbitrage/pairs.json")

	// find arbitrage cycle
	arbitragePath := models.ArbitragePath{
		ToCycle:   models.TransactionPath{},
		Cycle:     engine.FindArbitrage(pairsPtr, assetsPtr, indexPtr, configPtr.ReferenceAsset.Asset),
		FromCycle: models.TransactionPath{},
	}

	// check if arbitrage is reasonable
	if arbitragePath.Cycle != nil {

		expectedReturn := trade.CalculateExpectedReturn(arbitragePath.Cycle, pairsPtr)

		if !expectedReturn.Less(decimal.MustNew(1, 2)) {

			slog.Info("arbitrage opportunity found", "path", trade.GetSimplePath(arbitragePath.Cycle), "expectedReturn", expectedReturn, "lenPairs", len(*pairsPtr), "lenAssets", len(*assetsPtr))

			return true, arbitragePath, nil

		}
	}

	slog.Info("no arbitrage opportunity found")
	return false, models.ArbitragePath{}, nil
}

func main() {
	config, exchanges, clients, assets, index, err := initialization()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	interPairs, err := periodicUpdate(&config, &exchanges, &clients, &assets, &index)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var arbitragePath models.ArbitragePath
	var arbitrageFound bool

	for !arbitrageFound {
		arbitrageFound, arbitragePath, err = continuousUpdate(&config, &exchanges, &clients, &assets, &index, &interPairs)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// TODO: trade step
	fmt.Println(arbitragePath.Cycle)
}
