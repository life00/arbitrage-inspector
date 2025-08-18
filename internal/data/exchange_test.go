package data

import (
	"testing"

	"github.com/life00/arbitrage-inspector/internal/models"
)

func TestValidateExchanges(t *testing.T) {
	tests := []struct {
		name      string
		exchanges models.Exchanges
		wantErr   bool
	}{
		{
			name: "valid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "binance"},
					{Name: "kucoin"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "invalidexchange"},
				},
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{
					{Name: "binance"},
					{Name: "invalidexchange"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty exchanges",
			exchanges: models.Exchanges{
				Exchanges: []models.Exchange{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateExchanges(tt.exchanges); (err != nil) != tt.wantErr {
				t.Errorf("validateExchanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
