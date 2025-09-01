package main

import (
	"fmt"
	"log/slog"
	"os"

	// "github.com/govalues/decimal"
	"github.com/joho/godotenv"
	// "github.com/life00/arbitrage-inspector/internal/arbitrage"
	"github.com/life00/arbitrage-inspector/internal/data"
	// "github.com/life00/arbitrage-inspector/internal/models"
)

// main.go must be minimal with high abstraction
// all errors will be passed and handled here
// logging will be done where appropriate

func main() {
	// setup a default logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)
	slog.Info("successfully started logger")

	// get the environment variables (API credentials)
	err := godotenv.Load()
	if err != nil {
		slog.Error("failed to load .env file")
		os.Exit(1)
	}

	// TODO: Parse cli arguments and define inputs
	inputExchanges := []string{
		"binance",
		"kucoin",
		"bitget",
		// "htx",
		// "coinbase",
	}
	inputCurrencies := []string{
		"BTC",
		"ETH",
		"USDC",
		"DOGE",
		"SOL",
		"BNB",
		"USDT",
		"BCH",
		"LTC",
	}

	// 1. Data retrieval using data.go, exchange.go
	// 1.1. Validating and transforming the inputs; initializing the library
	exchanges, clients, err := data.InitializeExchanges(inputExchanges, inputCurrencies)
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

	err = saveExchanges(exchanges, "exchanges.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Transforming data

	// capital, err := decimal.NewFromFloat64(100)
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	//
	// assets, pairs := arbitrage.GenerateAssetPairs(&exchanges, capital)
	//
	// fmt.Println(pairs, assets)

	// 2.2. Bellman-Ford algorithm negative cycle detection

	// path, err := arbitrage.RunBellmanFord(graph, source)

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
