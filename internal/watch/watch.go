// Package watch provides Watcher interface to connect to WebSockets and update exchanges data.
package watch

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ccxt/ccxt/go/v4/pro"
	"github.com/life00/arbitrage-inspector/internal/models"
	"github.com/life00/arbitrage-inspector/internal/transform"
)

// --- Watcher ---

type Watcher struct {
	exchanges        *models.Exchanges
	clients          *models.Clients
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	exchangeWatchers map[string]*ExchangeWatcher
}

func NewWatcher(parentCtx context.Context, clients *models.Clients, exchanges *models.Exchanges, assetBalances *models.AssetBalances) *Watcher {
	ctx, cancel := context.WithCancel(parentCtx)
	w := &Watcher{
		exchanges:        exchanges,
		clients:          clients,
		ctx:              ctx,
		cancel:           cancel,
		exchangeWatchers: make(map[string]*ExchangeWatcher),
	}

	for id, client := range *clients {
		exData := (*exchanges)[id]
		var symbols []string
		for s := range exData.Markets {
			symbols = append(symbols, s)
		}

		if len(symbols) > 0 {
			w.exchangeWatchers[id] = newExchangeWatcher(id, client, symbols, assetBalances)
		}
	}
	return w
}

func (w *Watcher) Start() {
	slog.Info("starting watcher service", "exchanges", len(w.exchangeWatchers))
	for _, ew := range w.exchangeWatchers {
		ew.Start(w.ctx, &w.wg)
	}
}

func (w *Watcher) Stop() {
	slog.Info("stopping watcher service...")

	w.cancel()

	// forcefully close connections
	for id, ew := range w.exchangeWatchers {
		slog.Debug("closing exchange connection", "exchange", id)
		if errs := ew.client.Close(); len(errs) > 0 {
			// Just log warnings, we are shutting down anyway
			slog.Warn("errors closing exchange", "exchange", id, "errors", errors.Join(errs...))
		}
	}

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("watcher service stopped")
	case <-time.After(30 * time.Second):
		slog.Warn("watcher service shutdown timed out")
	}
}

func (w *Watcher) Sync() {
	var total, updated int
	for _, ew := range w.exchangeWatchers {
		total += ew.TotalSymbols()
		updated += ew.Sync(w.exchanges)
	}
	slog.Info("watcher sync complete", "coverage", fmt.Sprintf("%d/%d", updated, total))
}

func (w *Watcher) Status() {
	for id, ew := range w.exchangeWatchers {
		ew.Status(id)
	}
}

// --- ExchangeWatcher ---

type ExchangeWatcher struct {
	id            string
	client        ccxtpro.IExchange
	workers       []*Worker
	config        wsConfig
	assetBalances *models.AssetBalances
}

func newExchangeWatcher(id string, client ccxtpro.IExchange, symbols []string, assetBalances *models.AssetBalances) *ExchangeWatcher {
	cfg := getWSConfig(id)
	ew := &ExchangeWatcher{id: id, client: client, config: cfg, assetBalances: assetBalances}

	for i := 0; i < len(symbols); i += cfg.chunkSize {
		end := min(i+cfg.chunkSize, len(symbols))
		ew.workers = append(ew.workers, &Worker{
			id:      i/cfg.chunkSize + 1,
			symbols: symbols[i:end],
			client:  client,
			limit:   cfg.obLimit,
			cache:   make(map[string]ccxtpro.OrderBook),
		})
	}
	return ew
}

func (ew *ExchangeWatcher) Start(ctx context.Context, wg *sync.WaitGroup) {
	for i, wk := range ew.workers {
		wg.Add(1)
		go func(w *Worker, idx int) {
			if idx > 0 && ew.config.delay > 0 {
				select {
				case <-ctx.Done():
					wg.Done()
					return
				case <-time.After(time.Duration(idx*ew.config.delay) * time.Millisecond):
				}
			}
			w.Run(ctx, wg)
		}(wk, i)
	}
}

// OPTIMIZE: it might be possible to increase the speed of sync
// by performing CalculateEffectivePrices() concurrently
func (ew *ExchangeWatcher) Sync(globalExchanges *models.Exchanges) int {
	targetEx, ok := (*globalExchanges)[ew.id]
	if !ok {
		return 0
	}

	updated := 0
	assetBalances := *ew.assetBalances

	for _, wk := range ew.workers {
		wk.mu.Lock()

		for sym, raw := range wk.cache {
			if market, exists := targetEx.Markets[sym]; exists {
				base, _, _ := strings.Cut(sym, "/")

				market.Ask, market.Bid = transform.CalculateEffectivePrices(
					assetBalances[models.AssetKey{Exchange: ew.id, Currency: base}],
					raw,
				)

				if raw.Timestamp != nil {
					market.Timestamp = time.UnixMilli(*raw.Timestamp)
				}

				targetEx.Markets[sym] = market
				updated++
			}
		}

		// delete the cache
		wk.cache = make(map[string]ccxtpro.OrderBook)
		wk.connected = false

		wk.mu.Unlock()
	}
	return updated
}

func (ew *ExchangeWatcher) TotalSymbols() (count int) {
	for _, wk := range ew.workers {
		count += len(wk.symbols)
	}
	return
}

func (ew *ExchangeWatcher) Status(id string) {
	var active, total, updated int
	var totalDelay time.Duration
	now := time.Now()

	for _, wk := range ew.workers {
		wk.mu.RLock()
		total++
		if wk.connected {
			active++
		}
		for _, ob := range wk.cache {
			updated++
			if ob.Timestamp != nil && *ob.Timestamp > 0 {
				totalDelay += now.Sub(time.UnixMilli(*ob.Timestamp))
			}
		}
		wk.mu.RUnlock()
	}

	avgDelay := time.Duration(0)
	if updated > 0 {
		avgDelay = totalDelay / time.Duration(updated)
	}

	slog.Info("exchange watcher status",
		"id", id,
		"workers", fmt.Sprintf("%d/%d", active, total),
		"coverage", fmt.Sprintf("%d/%d", updated, ew.TotalSymbols()),
		"delay", avgDelay.Round(time.Millisecond),
	)
}

// --- Worker ---

type Worker struct {
	id         int
	symbols    []string
	client     ccxtpro.IExchange
	limit      int
	mu         sync.RWMutex
	cache      map[string]ccxtpro.OrderBook
	lastUpdate time.Time
	connected  bool
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	slog.Debug("started worker", "exchange", w.client.GetId(), "worker_id", w.id)
	for {
		// check context
		if ctx.Err() != nil {
			w.mu.Lock()
			w.connected = false
			w.mu.Unlock()
			return
		}

		ob, err := w.client.WatchOrderBookForSymbols(w.symbols, ccxtpro.WithWatchOrderBookForSymbolsLimit(int64(w.limit)))
		if err != nil {
			if ctx.Err() != nil {
				w.mu.Lock()
				w.connected = false
				w.mu.Unlock()
				return
			}

			w.mu.Lock()
			w.connected = false
			w.mu.Unlock()
			slog.Warn("worker error", "ex", w.client.GetId(), "id", w.id, "err", err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}
		w.updateCache(ob)
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
	w.cache[*ob.Symbol] = ob
}

// --- Utils ---

type wsConfig struct {
	chunkSize int // number of markets
	delay     int // spawn delay in ms
	obLimit   int // orderbook limit
}

// FIXME: some websockets are still getting broken because of rate limits.
// e.g. Kucoin and Bitmart are erroring, Bitmex and Binance ignoring
// It is necessary to further optimize these parameters and find how to
// overcome rate limits. Additionally it's necessary to have sufficient
// openbook limit, because transform.CalculateEffectivePrices is failing
// due to insufficient data.
func getWSConfig(id string) wsConfig {
	switch id {
	case "binance":
		return wsConfig{170, 30000, 10}
	case "kucoin":
		return wsConfig{50, 5000, 20}
	case "bitmart":
		return wsConfig{20, 5000, 10}
	case "bitmex":
		return wsConfig{50, 5000, 10}
	default:
		return wsConfig{50, 2000, 10}
	}
}
