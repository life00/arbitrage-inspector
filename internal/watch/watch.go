package watch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ccxt/ccxt/go/v4/pro"
	"github.com/govalues/decimal"
	"github.com/life00/arbitrage-inspector/internal/models"
)

type Watcher struct {
	exchanges *models.Exchanges
	clients   *models.Clients
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// active exchange watchers
	exchangeWatchers map[string]*ExchangeWatcher
}

// NewWatcher prepares the structure
func NewWatcher(parentCtx context.Context, clients *models.Clients, exchanges *models.Exchanges) *Watcher {
	ctx, cancel := context.WithCancel(parentCtx)

	w := &Watcher{
		exchanges:        exchanges,
		clients:          clients,
		ctx:              ctx,
		cancel:           cancel,
		exchangeWatchers: make(map[string]*ExchangeWatcher),
	}

	// Prepare data structures for each exchange
	for id, client := range *clients {
		// Retrieve markets to watch from the existing Exchanges data
		exData := (*exchanges)[id]

		// Extract symbol list
		var symbols []string
		for s := range exData.Markets {
			symbols = append(symbols, s)
		}

		if len(symbols) == 0 {
			continue
		}

		// Create the ExchangeWatcher and its Workers
		ew := newExchangeWatcher(id, client, symbols)
		w.exchangeWatchers[id] = ew
	}

	return w
}

// Start spins up all underlying ExchangeWatchers and their Workers.
func (w *Watcher) Start() {
	slog.Info("starting watcher service", "exchanges", len(w.exchangeWatchers))
	for _, ew := range w.exchangeWatchers {
		ew.Start(w.ctx, &w.wg)
	}
}

// FIXME:
// Stop sends the cancellation signal and waits for all workers to exit.
func (w *Watcher) Stop() {
	slog.Info("stopping watcher service...")
	w.cancel()
	w.wg.Wait()
	slog.Info("watcher service stopped")
}

// OPTIMIZE:
// Sync aggregates data from all workers, calculates effective prices, and updates the main model.
func (w *Watcher) Sync() {
	totalMarkets := 0
	totalUpdated := 0

	for _, ew := range w.exchangeWatchers {
		// Aggregate markets assigned to this watcher
		for _, worker := range ew.workers {
			totalMarkets += len(worker.symbols)
		}

		// Sync data and get the count of markets updated from cache
		totalUpdated += ew.Sync(w.exchanges)
	}

	slog.Info("watcher sync complete",
		"exchanges", len(w.exchangeWatchers),
		"coverage", fmt.Sprintf("%d/%d", totalUpdated, totalMarkets),
	)
}

// Status logs the health of all watchers and their connections.
func (w *Watcher) Status() {
	for id, ew := range w.exchangeWatchers {
		ew.LogStatus(id)
	}
}

type ExchangeWatcher struct {
	id       string
	client   ccxtpro.IExchange
	workers  []*Worker
	wsConfig webSocketConfig
}

func newExchangeWatcher(id string, client ccxtpro.IExchange, symbols []string) *ExchangeWatcher {
	ew := &ExchangeWatcher{
		id:     id,
		client: client,
	}
	ew.wsConfig = getWebSocketConfig(id)

	// Distribute symbols into chunks for concurrent workers
	// Using the specific chunkSize from the exchange configuration
	chunkSize := ew.wsConfig.chunkSize
	for i := 0; i < len(symbols); i += chunkSize {
		end := min(i+chunkSize, len(symbols))

		worker := &Worker{
			id:      i/chunkSize + 1,
			symbols: symbols[i:end],
			client:  client,
			limit:   ew.wsConfig.orderbookLimit,
			cache:   make(map[string]ccxtpro.OrderBook),
		}
		ew.workers = append(ew.workers, worker)
	}

	return ew
}

type webSocketConfig struct {
	chunkSize      int // number of symbols per worker
	spawnDelay     int // delay in milliseconds between spawning each worker
	orderbookLimit int // maximum number of orderbook entries
}

func getWebSocketConfig(exchangeID string) webSocketConfig {
	switch exchangeID {
	case "binance":
		return webSocketConfig{
			chunkSize:      170,
			spawnDelay:     30000,
			orderbookLimit: 3,
		}
	case "kucoin":
		return webSocketConfig{
			chunkSize:      50,
			spawnDelay:     5000,
			orderbookLimit: 5,
		}
	case "bitmart":
		return webSocketConfig{
			chunkSize:      20,
			spawnDelay:     5000,
			orderbookLimit: 5,
		}
	case "bitmex":
		return webSocketConfig{
			chunkSize:      50,
			spawnDelay:     5000,
			orderbookLimit: 10,
		}
	default:
		return webSocketConfig{
			chunkSize:      50,
			spawnDelay:     2000,
			orderbookLimit: 5,
		}
	}
}

func (ew *ExchangeWatcher) Start(ctx context.Context, wg *sync.WaitGroup) {
	// Calculate total market count for logging
	totalMarkets := 0
	for _, worker := range ew.workers {
		totalMarkets += len(worker.symbols)
	}

	slog.Info("starting exchange watcher",
		"id", ew.id,
		"markets", totalMarkets,
		"workers", len(ew.workers),
	)

	for i, worker := range ew.workers {
		wg.Add(1)

		// Spawn each worker in a goroutine that respects the spawnDelay
		go func(w *Worker, index int) {
			if index > 0 && ew.wsConfig.spawnDelay > 0 {
				delay := time.Duration(index*ew.wsConfig.spawnDelay) * time.Millisecond

				select {
				case <-ctx.Done():
					wg.Done()
					return
				case <-time.After(delay):
					// delay finished, proceed to Run
				}
			}
			w.Run(ctx, wg)
		}(worker, i)
	}
}

// Sync collects data from workers and updates the global model
func (ew *ExchangeWatcher) Sync(globalExchanges *models.Exchanges) int {
	// 1. Collect snapshots from all workers
	// We create a temporary map to hold the aggregated raw data
	snapshot := make(map[string]ccxtpro.OrderBook)
	updatedCount := 0

	for _, worker := range ew.workers {
		// Briefly lock the worker to copy its cache
		worker.mu.RLock()
		for sym, data := range worker.cache {
			snapshot[sym] = data
		}
		worker.mu.RUnlock()
	}

	// 2. Process data and update global model
	targetEx, ok := (*globalExchanges)[ew.id]
	if !ok {
		return 0
	}

	for symbol, rawData := range snapshot {
		if market, exists := targetEx.Markets[symbol]; exists {
			// Calculate effective price (considering liquidity/fees)
			effAsk, effBid := calculateEffectivePrice(rawData)

			market.Ask = effAsk
			market.Bid = effBid
			if rawData.Timestamp != nil {
				market.Timestamp = time.UnixMilli(*rawData.Timestamp)
			}

			// Update the market in the exchange copy
			targetEx.Markets[symbol] = market
			updatedCount++
		}
	}

	// 3. Save updated exchange back to the pointer
	(*globalExchanges)[ew.id] = targetEx

	return updatedCount
}

func (ew *ExchangeWatcher) LogStatus(exID string) {
	totalWorkers := 0
	activeWorkers := 0
	idleWorkers := 0

	totalMarketsAssigned := 0
	totalMarketsUpdated := 0

	var totalMarketDelay time.Duration
	now := time.Now()

	for _, w := range ew.workers {
		w.mu.RLock()
		totalWorkers++

		// Connection status
		if w.connected {
			activeWorkers++
		}
		// Check if data is stale (no updates in 30 seconds)
		if now.Sub(w.lastUpdate) > 30*time.Second {
			idleWorkers++
		}

		// Market Metrics
		totalMarketsAssigned += len(w.symbols)

		for _, ob := range w.cache {
			totalMarketsUpdated++

			// Calculate latency based on orderbook timestamp
			if ob.Timestamp != nil && *ob.Timestamp > 0 {
				marketTime := time.UnixMilli(*ob.Timestamp)
				delay := now.Sub(marketTime)
				if delay > 0 {
					totalMarketDelay += delay
				}
			}
		}
		w.mu.RUnlock()
	}

	avgMarketDelay := time.Duration(0)
	if totalMarketsUpdated > 0 {
		avgMarketDelay = totalMarketDelay / time.Duration(totalMarketsUpdated)
	}

	slog.Info("exchange watcher status",
		"id", exID,
		"workers", fmt.Sprintf("%d/%d", activeWorkers, totalWorkers),
		"workers_idle", idleWorkers,
		"coverage", fmt.Sprintf("%d/%d", totalMarketsUpdated, totalMarketsAssigned),
		"avg_delay", avgMarketDelay.Round(time.Millisecond).String(),
	)
}

type Worker struct {
	id      int
	symbols []string
	client  ccxtpro.IExchange
	limit   int

	// Local cache: Independent data structure per websocket
	cache map[string]ccxtpro.OrderBook
	mu    sync.RWMutex

	// Status tracking
	lastUpdate time.Time
	connected  bool
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	slog.Debug("worker started", "exchange", w.client.GetId(), "id", w.id, "markets", len(w.symbols))

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.connected = false
			w.mu.Unlock()
			return
		default:
			ob, err := w.client.WatchOrderBookForSymbols(
				w.symbols, ccxtpro.WithWatchOrderBookForSymbolsLimit(int64(w.limit)))
			if err != nil {
				// Handle Context Cancellation inside error
				if ctx.Err() != nil {
					return
				}

				w.mu.Lock()
				w.connected = false
				w.mu.Unlock()

				slog.Warn("websocket error", "id", w.id, "err", err)
				time.Sleep(5 * time.Second) // Backoff
				continue
			}

			// Update the local cache
			w.updateCache(ob)
		}
	}
}

func (w *Worker) updateCache(ob ccxtpro.OrderBook) {
	if ob.Symbol == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.connected = true
	w.lastUpdate = time.Now()

	// Overwrite or Add new entry
	// Note: We would convert CCXT Asks/Bids to our internal structure here
	// This is a simplified mapping
	w.cache[*ob.Symbol] = ob
}

// calculateEffectivePrice calculates price based on depth and capital.
// This is where you implement the "account for liquidity" logic.
func calculateEffectivePrice(ob ccxtpro.OrderBook) (ask, bid decimal.Decimal) {
	// TODO: Implement actual VWAP or depth-walking logic here
	// For now, we return the best price if available

	ask = decimal.Zero
	bid = decimal.Zero

	// Placeholder logic
	// if len(ob.Asks) > 0 { ask = ob.Asks[0].Price }
	// if len(ob.Bids) > 0 { bid = ob.Bids[0].Price }

	return ask, bid
}
