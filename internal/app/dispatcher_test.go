package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcher_postRunsOnConsumer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := NewDispatcher(8)
	var ran atomic.Int32
	go d.StartGeneric(ctx, func(f func()) { f() }) // test consumer just runs f
	for i := 0; i < 5; i++ {
		d.Post(func() { ran.Add(1) })
	}
	deadline := time.Now().Add(time.Second)
	for ran.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if ran.Load() != 5 {
		t.Fatalf("ran=%d want 5", ran.Load())
	}
}

func TestDispatcher_dropsNewestWhenFull(t *testing.T) {
	d := NewDispatcher(2)
	// Do NOT start a consumer — queue stays full.
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			d.Post(func() {})
		}()
	}
	wg.Wait()
	if d.Dropped() == 0 {
		t.Fatal("expected drops > 0")
	}
}

func TestDispatcher_recoversFromPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d := NewDispatcher(8)
	var ranAfter atomic.Int32
	go d.StartGeneric(ctx, func(f func()) {
		defer func() { _ = recover() }()
		f()
	})
	d.Post(func() { panic("boom") })
	d.Post(func() { ranAfter.Add(1) })
	deadline := time.Now().Add(time.Second)
	for ranAfter.Load() < 1 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if ranAfter.Load() != 1 {
		t.Fatal("post after panic did not run")
	}
}
