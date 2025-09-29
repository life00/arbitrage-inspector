package engine

import (
	"reflect"
	"testing"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestTranslatePath(t *testing.T) {
	btcUSDPath := []uint{0, 1, 2, 0}
	ethUSDPath := []uint{3, 4, 5, 3}

	index := models.Index{
		0: models.AssetKey{Exchange: "binance", Currency: "BTC"},
		1: models.AssetKey{Exchange: "binance", Currency: "USD"},
		2: models.AssetKey{Exchange: "binance", Currency: "ETH"},
		3: models.AssetKey{Exchange: "kraken", Currency: "ETH"},
		4: models.AssetKey{Exchange: "kraken", Currency: "USD"},
		5: models.AssetKey{Exchange: "kraken", Currency: "BTC"},
	}

	btcUSDPairs := models.TransactionPath{
		models.PairKey{From: index[0], To: index[1]},
		models.PairKey{From: index[1], To: index[2]},
		models.PairKey{From: index[2], To: index[0]},
	}

	ethUSDPairs := models.TransactionPath{
		models.PairKey{From: index[3], To: index[4]},
		models.PairKey{From: index[4], To: index[5]},
		models.PairKey{From: index[5], To: index[3]},
	}

	tests := []struct {
		name string
		path []uint
		want models.TransactionPath
	}{
		{
			name: "valid BTC/USD path",
			path: btcUSDPath,
			want: btcUSDPairs,
		},
		{
			name: "valid ETH/USD path",
			path: ethUSDPath,
			want: ethUSDPairs,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := translatePath(tc.path, &index)
			if !reflect.DeepEqual(actual, tc.want) {
				t.Errorf("translatePath(%v) got %v, want %v", tc.path, actual, tc.want)
			}
		})
	}
}
