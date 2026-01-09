package watch

import (
	"context"
	"sync"

	"github.com/life00/arbitrage-inspector/internal/models"
)

type Watcher struct {
	mu        sync.Mutex
	exchanges *models.Exchanges
}

func NewWatcher(watcherCtx context.Context, clients *models.Clients, exchanges *models.Exchanges) (watcher *Watcher) {
	watcher = &Watcher{}
	return watcher
}

func (w *Watcher) Start() {
}

func (w *Watcher) Sync() {
}

func (w *Watcher) Stop() {
}
