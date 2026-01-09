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
	slog.Info("starting initialization")

	// get the environment variables (API credentials)
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env file")
		return models.Config{}, nil, nil, nil, nil, err
	}

	// TODO: Parse cli arguments, config file, etc.
	// and define the config structure

	config := models.Config{
		Authenticate: false,
		Timeout:      60 * time.Second,
		Exchanges: []string{
			// "binance",
			// "kucoin",
			// "bitget",
			// "htx",
			// "coinbase",
			"binance", "bitfinex", "bitget", "bitmart", "bitmex", "bitstamp", "bitvavo", "blockchaincom", "bybit", "coinbase", "coincatch", "coinsph", "cryptocom", "foxbit", "gemini", "kraken", "lbank", "mexc", "okx", "phemex", "upbit", "wavesexchange", "whitebit", "zonda",
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
		ReferenceAsset: models.AssetBalance{
			Asset: models.AssetKey{
				Exchange: "binance",
				Currency: "USDC",
			},
			Balance: decimal.MustNew(2000, 0),
		},
		// all the assets where there is capital denominated in ReferenceAsset amount
		SourceAssets: map[models.AssetKey]models.AssetBalance{
			{Exchange: "binance", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "binance", Currency: "USDC"},
				Balance: decimal.Zero,
			},
			{Exchange: "kucoin", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "kucoin", Currency: "USDC"},
				Balance: decimal.Zero,
			},
			{Exchange: "bitget", Currency: "USDC"}: {
				Asset:   models.AssetKey{Exchange: "bitget", Currency: "USDC"},
				Balance: decimal.Zero,
			},
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
) (models.Pairs, error) {
	slog.Info("running periodic update")
	// update exchange data structure
	err := fetch.UpdateExchanges(exchangesPtr, clientsPtr, true, true, configPtr.Timeout)
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
	nominalAssetBalances := findAssetBalances(configPtr, exchangesPtr, assetsPtr)

	// create effective inter-exchange pairs
	slog.Debug("creating inter-exchange pairs")
	interPairs := createInterPairs(configPtr, exchangesPtr, assetsPtr, &nominalAssetBalances)

	return interPairs, nil
}

// finds source asset balances
func findAssetBalances(configPtr *models.Config, exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes) models.AssetBalances {
	slog.Debug("finding asset balances")
	// define pair configuration
	pairConfig := models.PairConfig{
		IntraType: models.FeeTypeNominal,
		InterType: models.FeeTypeNominal,
		Capital:   configPtr.ReferenceAsset.Balance,
	}
	pairs := make(models.Pairs)

	// transform: create nominal intra-exchange pairs (no fees)
	maps.Copy(pairs,
		transform.CreateIntraExchangePairs(pairConfig, exchangesPtr, assetsPtr))

	// transform: create nominal inter-exchange pairs (no fees)
	maps.Copy(pairs,
		transform.CreateInterExchangePairs(pairConfig, exchangesPtr, assetsPtr, &models.AssetBalances{}))
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

	// engine: find balances of all assets
	assetBalances := engine.FindBalances(&pairs, assetsPtr, configPtr.ReferenceAsset)

	// transform: config.SourceAssets balances
	for key := range configPtr.SourceAssets {
		configPtr.SourceAssets[key] = assetBalances[key]
	}

	// TODO:
	// trade: ensure sufficient balances

	return assetBalances
}

// creates effective inter-exchange pairs
func createInterPairs(configPtr *models.Config, exchangesPtr *models.Exchanges, assetsPtr *models.AssetIndexes, nominalAssetBalancesPtr *models.AssetBalances) models.Pairs {
	// initialize pair configuration
	pairConfig := models.PairConfig{
		IntraType:   models.FeeTypeEffective,
		InterType:   models.FeeTypeConstant,
		ConstantFee: decimal.MustNew(1, 0),
		Capital:     configPtr.ReferenceAsset.Balance,
	}
	pairs := make(models.Pairs)

	// transform: create effective intra-exchange pairs (with regular bid/ask prices)
	maps.Copy(pairs,
		transform.CreateIntraExchangePairs(pairConfig, exchangesPtr, assetsPtr))

	// transform: create effective inter-exchange pairs (with some constant amount denominated in ReferenceAsset like 1 USDT)
	maps.Copy(pairs,
		transform.CreateInterExchangePairs(pairConfig, exchangesPtr, assetsPtr, nominalAssetBalancesPtr))

	// engine: find balances of all assets
	effectiveAssetBalances := engine.FindBalances(&pairs, assetsPtr, configPtr.ReferenceAsset)

	// transform: create actual effective inter-exchange pairs (using all asset balances)
	pairConfig = models.PairConfig{
		InterType: models.FeeTypeEffective,
	}
	interPairs := transform.CreateInterExchangePairs(pairConfig, exchangesPtr, assetsPtr, &effectiveAssetBalances)

	return interPairs
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
	slog.Info("running continuous update")
	// TODO:
	// client: wait some time
	// watch: call watcher to update data
	// transform: create actual effective inter-exchange pairs
	// engine: search for reasonable arbitrage and find full ArbitragePath
	// trade: arbitrage is profitable?

	// TODO: watcher
	// watch: initialize orderbook watcher (establish websocket connections)
	// watch: cache all received orderbook data
	// transform: calculate effective prices (from orderbook data)
	// watch: update exchanges data structure

	// FIXME: temporary solution

	// wait some time before each update
	time.Sleep(30 * time.Second)

	// update exchange price data using regular bid/ask prices
	err := fetch.UpdateExchanges(exchangesPtr, clientsPtr, false, false, configPtr.Timeout)
	if err != nil {
		return false, models.ArbitragePath{}, nil
	}

	// calculate regular intra-exchange pairs
	pairConfig := models.PairConfig{
		IntraType: models.FeeTypeEffective,
	}
	slog.Debug("creating intra-exchange pairs")
	intraPairs := transform.CreateIntraExchangePairs(pairConfig, exchangesPtr, assetsPtr)
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
		Cycle:     engine.FindArbitrage(pairsPtr, assetsPtr, indexPtr, configPtr.SourceAssets),
		FromCycle: models.TransactionPath{},
	}

	// check if arbitrage is reasonable
	if arbitragePath.Cycle != nil {

		expectedReturn := trade.CalculateExpectedReturn(arbitragePath.Cycle, pairsPtr)
		simplePath := trade.GetSimplePath(arbitragePath.Cycle)

		if !expectedReturn.Less(decimal.MustNew(1, 2)) && len(simplePath) < 8 {

			slog.Info("arbitrage opportunity found", "path", simplePath, "expectedReturn", expectedReturn, "lenPairs", len(*pairsPtr), "lenAssets", len(*assetsPtr))

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

	interPairs, err := periodicUpdate(&config, &exchanges, &clients, &assets)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var arbitragePath models.ArbitragePath
	var arbitrageFound bool

	slog.Info("starting continuous update")
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
