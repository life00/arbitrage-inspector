package main

import (
	"fmt"
	"log/slog"
	"os"
	// "time"

	"github.com/govalues/decimal"
	// "github.com/joho/godotenv"
	// "github.com/life00/arbitrage-inspector/internal/fetch"
	"github.com/life00/arbitrage-inspector/internal/models"
	// "github.com/life00/arbitrage-inspector/internal/trade"
	"github.com/life00/arbitrage-inspector/internal/transform"
	"github.com/lmittmann/tint"
)

// main.go must be minimal with high abstraction
// all errors will be passed and handled here
// logging will be done where appropriate

func main() {
	// setup a default logger
	logger := slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			// AddSource: true,
			Level: slog.LevelDebug,
		}),
	)
	slog.SetDefault(logger)

	// get the environment variables (API credentials)
	// err := godotenv.Load()
	// if err != nil {
	// 	slog.Error("failed to load .env file")
	// 	os.Exit(1)
	// }

	// TODO: Parse cli arguments and define inputs

	var config models.Config

	// config = models.Config{
	// 	Exchanges: []string{
	// 		"binance",
	// 		"kucoin",
	// 		"bitget",
	// 		// "htx",
	// 		// "coinbase",
	// 	},
	// 	CurrencyInputMode: models.AllCurrencies,
	// 	Currencies: []string{
	// 		"BTC",
	// 		"ETH",
	// 		"USDC",
	// 		"DOGE",
	// 		"SOL",
	// 		"BNB",
	// 		"USDT",
	// 		"BCH",
	// 		"LTC",
	// 		"XMR",
	// 	},
	// 	ExcludedCurrencies: []string{
	// 		// problematic currency codes
	// 		"NEIRO",
	// 		"BROCCOLI",
	// 	},
	// }

	// 1. Data retrieval using data.go, exchange.go
	// 1.1. Validating and transforming the inputs; initializing the library
	// exchanges, clients, err := data.InitializeExchanges(config)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// 1.2. Fetching price data and fees

	// err = data.UpdateExchanges(&exchanges, &clients, true, true)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	// saveAnyJson(exchanges, "/home/user/dev/src/arbitrage/exchanges.json")

	// load data from cached exchanges.json
	exchanges, err := loadAnyJson[models.Exchanges]("/home/user/dev/src/arbitrage/exchanges.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Transforming data

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config = models.Config{
		ReferenceAsset: models.AssetBalance{
			Asset: models.AssetKey{
				Exchange: "binance",
				Currency: "USDC",
			},
			Balance: decimal.MustNew(1000000, 0),
		},
		SourceAssets: map[models.AssetKey]models.AssetBalance{
			{Exchange: "binance", Currency: "USDC"}: {
				Asset: models.AssetKey{Exchange: "binance", Currency: "USDC"},
				// balance is not defined because it will be generated further
			},
			{Exchange: "kucoin", Currency: "USDC"}: {
				Asset: models.AssetKey{Exchange: "kucoin", Currency: "USDC"},
			},
			{Exchange: "bitget", Currency: "USDC"}: {
				Asset: models.AssetKey{Exchange: "bitget", Currency: "USDC"},
			},
		},
	}

	config.SourceAssets = transform.FindAssetBalances(config, &exchanges)

	// 1. initialize exchanges data structure & update it
	// 2. figure out the balances of assets in SourceAssets data structure
	// 2.1. create currency pairs without any fees
	// 2.2. create a nominal graph
	// 2.3. run bellman-ford
	// 2.4. use resulting weights to convert nominal value of reference asset to all source assets
	// 3. create intra-exchange pairs
	// 4. create inter-exchange pairs
	// 4.1. use previously created intra pairs and create inter pairs with a  constant 1-2 USD for all fees (whenever the destination currency is on a different exchange than the reference currency), it is used to roughly estimate real balances in all currencies
	// 4.2. create a graph for balance estimation
	// 4.3. run bellman-ford
	// 4.4. use the resulting weights to calculate balances for all possible currencies
	// 4.5. use the calculated balances in the actual inter fee estimation and pair creation
	// 4.*. ...
	// 5. combine with inter-exchange pairs
	// 6. run the main algorithm, etc.

	// pairs, assets, index := transform.CreateAssetPairs(&exchanges, config.Capital)
	//
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

	// path := arbitrage.FindArbitrage(&pairs, &assets, &index, config.SourceAsset)

	// if path == nil {
	// 	slog.Info("no arbitrage opportunity found")
	// } else {
	// 	expectedReturn := trade.CalculateExpectedReturn(path, &pairs)
	//
	// 	slog.Info("arbitrage opportunity found", "path", trade.GetSimplePath(path), "expectedReturn", expectedReturn)
	// }
	//
	// slog.Info("starting update loop")
	//
	// for {
	// 	time.Sleep(time.Second * 10)
	// 	err = data.UpdateExchanges(&exchanges, &clients, false, false)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		os.Exit(1)
	// 	}
	// 	pairs, assets, index = arbitrage.CreateAssetPairs(&exchanges, config.Capital)
	// 	path = arbitrage.FindArbitrage(&pairs, &assets, &index, config.SourceAsset)
	//
	// 	if path == nil {
	// 		slog.Info("no arbitrage opportunity found")
	// 	} else {
	// 		expectedReturn := trade.CalculateExpectedReturn(path, &pairs)
	//
	// 		slog.Info("arbitrage opportunity found", "path", trade.GetSimplePath(path), "expectedReturn", expectedReturn, "lenPairs", len(pairs), "lenAssets", len(assets))
	// 	}
	//
	// }

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
