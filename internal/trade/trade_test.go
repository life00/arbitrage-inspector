package trade

import (
	"testing"

	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestCalculateExpectedReturn(t *testing.T) {
	// common assets and pairs
	binanceBTC := models.AssetKey{Exchange: "Binance", Currency: "BTC"}
	binanceETH := models.AssetKey{Exchange: "Binance", Currency: "ETH"}
	binanceUSD := models.AssetKey{Exchange: "Binance", Currency: "USD"}

	ethBTCKey := models.PairKey{From: binanceETH, To: binanceBTC}
	btcUSDKey := models.PairKey{From: binanceBTC, To: binanceUSD}

	ethBTCWeight, _ := decimal.NewFromFloat64(1.1)
	btcUSDWeight, _ := decimal.NewFromFloat64(1.05)

	pairsData := models.Pairs{
		ethBTCKey: models.Pair{
			Weight: ethBTCWeight,
		},
		btcUSDKey: models.Pair{
			Weight: btcUSDWeight,
		},
	}

	tests := []struct {
		name     string
		path     models.TransactionPath
		pairs    models.Pairs
		expected decimal.Decimal
	}{
		{
			name:     "valid two-pair path",
			path:     models.TransactionPath{ethBTCKey, btcUSDKey},
			pairs:    pairsData,
			expected: func() decimal.Decimal { d, _ := decimal.NewFromFloat64(0.155); return d }(), // (1.1 * 1.05) - 1 = 1.155 - 1 = 0.155
		},
		{
			name:     "empty path",
			path:     models.TransactionPath{},
			pairs:    pairsData,
			expected: func() decimal.Decimal { d, _ := decimal.NewFromFloat64(0); return d }(), // 1 - 1 = 0
		},
		{
			name:     "missing pair in path",
			path:     models.TransactionPath{ethBTCKey, models.PairKey{From: binanceBTC, To: binanceETH}},
			pairs:    pairsData,
			expected: func() decimal.Decimal { d, _ := decimal.NewFromInt64(0, 0, 0); return d }(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := CalculateExpectedReturn(tc.path, &tc.pairs)

			if !actual.Equal(tc.expected) {
				t.Errorf("CalculateExpectedReturn(%+v) returned %v, expected %v", tc.path, actual, tc.expected)
			}
		})
	}
}
