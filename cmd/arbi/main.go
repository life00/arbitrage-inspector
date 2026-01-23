package main

import (
	"context"
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
	"github.com/life00/arbitrage-inspector/internal/watch"
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
			"bitget", "bitmart", "bitmex", "kucoin",
		},
		CurrencyInputMode: models.SpecifiedCurrencies,
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
			"ZKP",
			"LOT",
			"EUR",
		},
		ReferenceAsset: models.AssetBalance{
			Asset: models.AssetKey{
				Exchange: "bitget",
				Currency: "USDC",
			},
			Balance: decimal.MustNew(100, 0),
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
) (models.Pairs, models.AssetBalances, error) {
	slog.Info("running periodic update")
	// update exchange data structure
	err := fetch.UpdateExchanges(exchangesPtr, clientsPtr, true, true, configPtr.Timeout)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}
	// saveAnyJSON(*exchangesPtr, "/home/user/dev/src/arbitrage/exchanges.json")
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

	return interPairs, nominalAssetBalances, nil
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
	assetsPtr *models.AssetIndexes,
	indexPtr *models.Index,
	pairsPtr *models.Pairs,
	w *watch.Watcher,
) (bool, models.ArbitragePath, error) {
	time.Sleep(10 * time.Second)
	slog.Info("running continuous update")

	// watch: call watcher to update data
	w.Status()
	w.Sync()

	// transform: create actual effective inter-exchange pairs

	// calculate regular intra-exchange pairs
	pairConfig := models.PairConfig{
		IntraType: models.FeeTypeEffective,
	}
	slog.Debug("creating intra-exchange pairs")
	intraPairs := transform.CreateIntraExchangePairs(pairConfig, exchangesPtr, assetsPtr)
	maps.Copy(*pairsPtr, intraPairs)

	// TODO:
	// engine: search for reasonable arbitrage and find full ArbitragePath
	arbitragePath := models.ArbitragePath{
		ToCycle:   models.TransactionPath{},
		Cycle:     engine.FindArbitrage(pairsPtr, assetsPtr, indexPtr, configPtr.SourceAssets),
		FromCycle: models.TransactionPath{},
	}

	// TODO:
	// trade: arbitrage is profitable?
	if arbitragePath.Cycle != nil {

		expectedReturn := trade.CalculateExpectedReturn(arbitragePath.Cycle, pairsPtr)
		simplePath := trade.GetSimplePath(arbitragePath.Cycle)

		// this limits the arbitrage opportunities to only high-value ones
		// if !expectedReturn.Less(decimal.MustNew(1, 2)) && len(simplePath) < 8 {
		if true {
			slog.Info("arbitrage opportunity found", "path", simplePath, "expectedReturn", expectedReturn, "lenPairs", len(*pairsPtr), "lenAssets", len(*assetsPtr))

			return true, arbitragePath, nil

		}
	}

	slog.Info("no arbitrage opportunity found")
	return false, models.ArbitragePath{}, nil
}

// verifyArbitrage() fetches orderbook of intra-exchange pairs, computes VWAP effective bid/ask prices
// and checks if the arbitrage is still profitable
func verifyArbitrage(
	arbitragePath models.ArbitragePath,
	clientsPtr *models.Clients,
	nominalAssetBalancesPtr *models.AssetBalances,
	exchangesPtr *models.Exchanges,
	interPairsPtr *models.Pairs,
	assetsPtr *models.AssetIndexes,
) bool {
	// transform: get transaction markets from ArbitragePath
	localExchanges := transform.GetTransactionMarkets(arbitragePath, exchangesPtr)
	assetBalances := *nominalAssetBalancesPtr

	// fetch: fetch orderbook data of markets in ArbitragePath
	err := fetch.UpdateOrderBooks(&localExchanges, clientsPtr)
	if err != nil {
		slog.Warn("error fetching orderbooks", "error", err)
		return false
	}

	// transform: calculate volume-weighted average prices
	for eID, exchange := range localExchanges {
		for mID, market := range exchange.Markets {
			market.Bid, market.Ask = transform.CalculateEffectivePrices(
				assetBalances[models.AssetKey{Exchange: exchange.ID, Currency: market.Base}],
				market.OrderBook,
				eID,
				mID,
			)
			localExchanges[eID].Markets[mID] = market
		}
	}

	// tranform: recreate pairs
	pairConfig := models.PairConfig{
		IntraType: models.FeeTypeEffective,
	}
	pairs := make(models.Pairs)
	maps.Copy(pairs, *interPairsPtr)

	maps.Copy(pairs,
		transform.CreateIntraExchangePairs(pairConfig, &localExchanges, assetsPtr),
	)

	// trade: check if arbitrage is still profitable
	expectedReturn := trade.CalculateExpectedReturn(arbitragePath.Cycle, &pairs)

	if expectedReturn.IsNeg() {
		slog.Warn("arbitrage has insufficient liquidity", "expectedReturn", expectedReturn)
		return false
	} else {
		slog.Info("arbitrage has sufficient liquidity", "expectedReturn", expectedReturn)
		return true
	}
}

func main() {
	config, exchanges, clients, assets, index, err := initialization()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	interPairs, nominalAssetBalances, err := periodicUpdate(&config, &exchanges, &clients, &assets)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var arbitragePath models.ArbitragePath
	var arbitrageFound bool

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create and start watcher
	w := watch.NewWatcher(ctx, &clients, &exchanges, &nominalAssetBalances)
	w.Start()
	// client: wait some time to initialize all WebSockets
	time.Sleep(1 * time.Minute)

	for !arbitrageFound {
		arbitrageFound, arbitragePath, err = continuousUpdate(&config, &exchanges, &assets, &index, &interPairs, w)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// verify arbitrage liquidity of the found arbitrage
	if verifyArbitrage(arbitragePath, &clients, &nominalAssetBalances, &exchanges, &interPairs, &assets) {
		// TODO: implement trade execution
		slog.Info("executing trade")
	}
}
