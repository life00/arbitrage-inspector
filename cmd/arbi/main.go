package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/govalues/decimal"
	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/arbitrage"
	"github.com/life00/arbitrage-inspector/internal/data"
	"github.com/life00/arbitrage-inspector/internal/models"
	"github.com/life00/arbitrage-inspector/internal/trade"
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
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env file")
		os.Exit(1)
	}

	// TODO: Parse cli arguments and define inputs

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

	// 1. Data retrieval using data.go, exchange.go
	// 1.1. Validating and transforming the inputs; initializing the library
	exchanges, clients, err := data.InitializeExchanges(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 1.2. Fetching price data and fees

	err = data.UpdateExchanges(&exchanges, &clients, true, true)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	saveAnyJson(exchanges, "/home/user/dev/src/arbitrage/exchanges.json")

	// load data from cached exchanges.json
	// exchanges, err := loadAnyJson[models.Exchanges]("/home/user/dev/src/arbitrage/exchanges.json")
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Transforming data

	config.Capital, err = decimal.NewFromFloat64(1000000)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pairs, assets, index := arbitrage.CreateAssetPairs(&exchanges, config.Capital)

	type SerializedPair struct {
		Key   models.PairKey
		Value models.Pair
	}
	var serializedPairs []SerializedPair
	for key, value := range pairs {
		serializedPairs = append(serializedPairs, SerializedPair{
			Key:   key,
			Value: value,
		})
	}
	saveAnyJson(serializedPairs, "/home/user/dev/src/arbitrage/pairs.json")

	// 2.2. Bellman-Ford algorithm negative cycle detection
	config.SourceAsset = models.AssetKey{Exchange: "binance", Currency: "USDC"}

	path := arbitrage.FindArbitrage(&pairs, &assets, &index, config.SourceAsset)

	if path == nil {
		slog.Info("no arbitrage opportunity found")
	} else {
		expectedReturn := trade.CalculateExpectedReturn(path, &pairs)

		slog.Info("arbitrage opportunity found", "path", trade.GetSimplePath(path), "expectedReturn", expectedReturn)
	}

	slog.Info("starting update loop")

	for {
		time.Sleep(time.Second * 10)
		err = data.UpdateExchanges(&exchanges, &clients, false, false)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pairs, assets, index = arbitrage.CreateAssetPairs(&exchanges, config.Capital)
		path = arbitrage.FindArbitrage(&pairs, &assets, &index, config.SourceAsset)

		if path == nil {
			slog.Info("no arbitrage opportunity found")
		} else {
			expectedReturn := trade.CalculateExpectedReturn(path, &pairs)

			slog.Info("arbitrage opportunity found", "path", trade.GetSimplePath(path), "expectedReturn", expectedReturn, "lenPairs", len(pairs), "lenAssets", len(assets))
		}

	}

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
