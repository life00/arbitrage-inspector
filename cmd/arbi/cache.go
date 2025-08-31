package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func saveExchanges(exchanges models.Exchanges, filename string) error {
	data, err := json.MarshalIndent(exchanges, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal JSON: %w", err)
		slog.Error(err.Error())
		return err
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write to file: %w", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("exchanges saved successfully", "filename", filename, "size_bytes", len(data))
	return nil
}

func loadExchanges(filename string) (models.Exchanges, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to read file: %w", err)
		slog.Error(err.Error())
		return nil, err
	}

	var exchanges models.Exchanges
	err = json.Unmarshal(data, &exchanges)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal JSON: %w", err)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("exchanges loaded successfully", "filename", filename, "num_exchanges", len(exchanges))
	return exchanges, nil
}
