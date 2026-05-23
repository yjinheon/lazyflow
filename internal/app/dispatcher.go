package app

import (
	"context"
	"sync/atomic"

	"github.com/rivo/tview"
)

// Dispatcher serializes all UI mutations onto a single consumer goroutine,
// preventing the QueueUpdateDraw re-entrancy deadlock that affected earlier
// versions of lazyflow (see commit 7805477).
//
// Rules (load-bearing):
//   1. Never call tviewApp.QueueUpdateDraw directly. Use Post.
//   2. The closure passed to Post must read fresh state from the store —
//      do not capture stale data. Drops are then harmless.
//   3. Callbacks inside store.Subscribe must be non-blocking (Post or short).
type Dispatcher struct {
	ch      chan func()
	dropped atomic.Uint64
}

func NewDispatcher(cap int) *Dispatcher {
	return &Dispatcher{ch: make(chan func(), cap)}
}

// Post enqueues f for execution on the UI thread. Non-blocking; if the queue
// is full, the newest post is dropped and Dropped() is incremented.
func (d *Dispatcher) Post(f func()) {
	select {
	case d.ch <- f:
	default:
		d.dropped.Add(1)
	}
}

func (d *Dispatcher) Dropped() uint64 { return d.dropped.Load() }

// Start consumes the queue, invoking each f via app.QueueUpdateDraw with
// panic recovery. Blocks until ctx is done.
func (d *Dispatcher) Start(ctx context.Context, app *tview.Application) {
	d.StartGeneric(ctx, func(f func()) {
		defer func() { _ = recover() }()
		app.QueueUpdateDraw(f)
	})
}

// StartGeneric is exported for tests so we can substitute the tview call.
func (d *Dispatcher) StartGeneric(ctx context.Context, exec func(func())) {
	for {
		select {
		case <-ctx.Done():
			return
		case f := <-d.ch:
			exec(f)
		}
	}
}
