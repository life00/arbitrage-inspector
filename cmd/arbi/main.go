package main

import (
	"log/slog"
	"os"
)

// main.go must be minimal with high abstraction
// all errors will be passed and handled here
// logging will be done where appropriate

func main() {
	// setup a default logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("Successfully initialized logger")

	// parse cli arguments

	// TODO: 1. Data retrieval using data.go and fees.go

	// TODO: 2. Arbitrage identification using arbitrage.go

	// TODO: 3. Trade execution using trade.go
}
