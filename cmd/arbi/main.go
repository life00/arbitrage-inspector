package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/data"
	"github.com/life00/arbitrage-inspector/internal/models"
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
	exchanges := models.Exchanges{
		Exchanges: []models.Exchange{
			{Name: "binance"},
			{Name: "kucoin"},
		},
	}

	currencies := models.Currencies{
		Currencies: []models.Currency{
			{Id: "BTC"},
			{Id: "ETH"},
			{Id: "USDC"},
		},
	}

	// TODO: Define data structures

	// 1. Data retrieval using data.go, exchange.go
	slog.Info("fetching data...")
	err = data.FetchData(exchanges, currencies)
	if err != nil {
		panic(err)
	}

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Graph creation (with fees)
	// 2.2. Bellman-Ford algorithm negative cycle detection
	// 2.3. Arbitrage path retrieval

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
