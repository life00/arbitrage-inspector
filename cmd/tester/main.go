package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/govalues/decimal"
	"github.com/joho/godotenv"
	"github.com/life00/arbitrage-inspector/internal/fetch"
	"github.com/life00/arbitrage-inspector/internal/models"
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
			"backpack",
			"bitget",
			"bitmart",
			"bitmex",
			"coinex",
			"kucoin",
			"toobit",
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
		},
		ReferenceAsset: models.AssetBalance{
			Asset: models.AssetKey{
				Exchange: "bitget",
				Currency: "USDC",
			},
			Balance: decimal.MustNew(10000, 0),
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

func main() {
	_, exchanges, clients, _, _, err := initialization()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := watch.NewWatcher(ctx, &clients, &exchanges)
	w.Start()

	time.Sleep(20 * time.Second)
	w.Status()
	time.Sleep(10 * time.Second)

	start := time.Now()
	w.Sync()
	fmt.Println(time.Since(start))

	w.Stop()
	time.Sleep(1 * time.Minute)
}
