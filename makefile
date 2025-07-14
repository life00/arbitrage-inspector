build: cmd/arbi/main.go internal/arbitrage/arbitrage.go internal/data/data.go internal/exchange/exchange.go internal/fees/fees.go internal/models/models.go internal/trade/trade.go
	go build -o arbi cmd/arbi/main.go
