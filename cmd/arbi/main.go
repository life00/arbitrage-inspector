package main

import (
	"log/slog"
	"os"
	// "github.com/life00/arbitrage-inspector/internal/models"
)

// main.go must be minimal with high abstraction
// all errors will be passed and handled here
// logging will be done where appropriate

func main() {
	// setup a default logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Successfully initialized logger")

	// TODO: Parse cli arguments and define imputs

	// TODO: Define data structures

	// 1. Data retrieval using data.go
	// 1.1. Using exchange.go with CCXT
	// 1.2. Using fees.go

	// 2. Arbitrage identification using arbitrage.go
	// 2.1. Graph creation (with fees)
	// 2.2. Bellman-Ford algorithm negative cycle detection
	// 2.3. Arbitrage path retrieval

	// 3. Trade execution using trade.go
	// 3.1. While the arbitrage is still present continue the trading cycle (check using data.go)
}
