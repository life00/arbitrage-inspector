package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
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
	}

	// TODO: Parse cli arguments and define inputs
	exchanges := []string{
		"binance",
		"kucoin",
		"bitget",
		"htx",
	}
	currencies := []string{
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

	// TODO: Define data structures

	// 1. Data retrieval using data.go, exchange.go
	// 1.1. Validating and transforming the inputs; initializing the library
	slog.Info("initializing data...")
	data, clients, err := data.InitializeData(exchanges, currencies)
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(data, clients)

	// 1.2. Fetching price data and fees

	// data, err := data.UpdateData(&data)

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Graph creation

	// graph, err := arbitrage.InitializeGraph(data)

	// 2.2. Bellman-Ford algorithm negative cycle detection

	// path, err := arbitrage.RunBellmanFord(graph, source)

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
