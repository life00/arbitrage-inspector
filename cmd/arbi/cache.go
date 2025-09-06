package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

func saveAnyJson(data any, filename string) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		err = fmt.Errorf("failed to marshal JSON: %w", err)
		slog.Error(err.Error())
		return err
	}

	err = os.WriteFile(filename, file, 0644)
	if err != nil {
		err = fmt.Errorf("failed to write to file: %w", err)
		slog.Error(err.Error())
		return err
	}

	slog.Info("data saved successfully", "filename", filename, "size_bytes", len(file))
	return nil
}

func loadAnyJson[T any](filename string) (T, error) {
	var data T

	file, err := os.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("failed to read file: %w", err)
		slog.Error(err.Error())
		return data, err
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal JSON: %w", err)
		slog.Error(err.Error())
		return data, err
	}

	slog.Info("data loaded successfully", "filename", filename)
	return data, nil
}
