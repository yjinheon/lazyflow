package app

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/yjinheon/lazyflow/internal/debugutil"
)

// Poller manages periodic data fetching with context-based lifecycle.
type Poller struct {
	ctx    context.Context
	cancel context.CancelFunc

	// mu protects subCancels. Restart is normally called from the tview main
	// goroutine, but defending the map costs nothing and prevents a future
	// caller from introducing a silent race.
	mu         sync.Mutex
	subCancels map[string]context.CancelFunc
}

func NewPoller(parent context.Context) *Poller {
	ctx, cancel := context.WithCancel(parent)
	return &Poller{
		ctx:        ctx,
		cancel:     cancel,
		subCancels: make(map[string]context.CancelFunc),
	}
}

// Fixed starts a polling loop that runs for the lifetime of the poller.
func (p *Poller) Fixed(interval time.Duration, immediate bool, fn func(ctx context.Context)) {
	go func() {
		// ±15% jitter on first tick to avoid startup thundering-herd
		// when multiple Fixed pollers spin up together.
		jitter := time.Duration(float64(interval) * 0.15 * (rand.Float64()*2 - 1))
		debugutil.Tag("FZ-poll", "Fixed loop START interval=%v immediate=%v jitter=%v", interval, immediate, jitter)
		if immediate {
			tStart := time.Now()
			fn(p.ctx)
			debugutil.Tag("FZ-poll", "Fixed immediate fn END elapsed=%v", time.Since(tStart))
		}
		if jitter > 0 {
			time.Sleep(jitter)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				debugutil.Tag("FZ-poll", "Fixed loop EXIT")
				return
			case <-ticker.C:
				tStart := time.Now()
				fn(p.ctx)
				if d := time.Since(tStart); d > 500*time.Millisecond {
					debugutil.Tag("FZ-poll", "Fixed fn slow elapsed=%v interval=%v", d, interval)
				}
			}
		}
	}()
}

// Restart cancels any existing sub-poller with the given name and starts a new one.
// Use this for polls that change target when a selection changes (e.g. runs for a DAG).
func (p *Poller) Restart(name string, interval time.Duration, fn func(ctx context.Context)) {
	debugutil.Tag("FZ-poll", "Restart name=%s interval=%v", name, interval)

	p.mu.Lock()
	if cancel, ok := p.subCancels[name]; ok {
		debugutil.Tag("FZ-poll", "Restart %s cancelling previous goroutine", name)
		cancel()
	}
	subCtx, subCancel := context.WithCancel(p.ctx)
	p.subCancels[name] = subCancel
	p.mu.Unlock()

	go func() {
		debugutil.Tag("FZ-poll", "sub-poller %s START", name)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-subCtx.Done():
				debugutil.Tag("FZ-poll", "sub-poller %s EXIT", name)
				return
			case <-ticker.C:
				tStart := time.Now()
				fn(subCtx)
				if d := time.Since(tStart); d > 500*time.Millisecond {
					debugutil.Tag("FZ-poll", "sub-poller %s fn slow elapsed=%v", name, d)
				}
			}
		}
	}()
}

// Stop cancels all polling.
func (p *Poller) Stop() {
	debugutil.Tag("FZ-poll", "Stop")
	p.cancel()
}

// StopSub cancels a single named sub-poller without affecting others.
// Use this when a poll is no longer relevant (e.g., user left the tab).
func (p *Poller) StopSub(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cancel, ok := p.subCancels[name]; ok {
		debugutil.Tag("FZ-poll", "StopSub %s", name)
		cancel()
		delete(p.subCancels, name)
	}
}
