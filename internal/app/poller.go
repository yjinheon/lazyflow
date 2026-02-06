package app

import (
	"context"
	"time"
)

// Poller manages periodic data fetching with context-based lifecycle.
type Poller struct {
	ctx    context.Context
	cancel context.CancelFunc

	// dynamic sub-pollers that restart on selection changes
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
		if immediate {
			fn(p.ctx)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				fn(p.ctx)
			}
		}
	}()
}

// Restart cancels any existing sub-poller with the given name and starts a new one.
// Use this for polls that change target when a selection changes (e.g. runs for a DAG).
func (p *Poller) Restart(name string, interval time.Duration, fn func(ctx context.Context)) {
	if cancel, ok := p.subCancels[name]; ok {
		cancel()
	}
	subCtx, subCancel := context.WithCancel(p.ctx)
	p.subCancels[name] = subCancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-subCtx.Done():
				return
			case <-ticker.C:
				fn(subCtx)
			}
		}
	}()
}

// Stop cancels all polling.
func (p *Poller) Stop() {
	p.cancel()
}
