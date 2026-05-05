package debugutil

import (
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on http.DefaultServeMux
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func GoID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	if i := strings.IndexByte(s, ' '); i > 0 {
		id, _ := strconv.ParseUint(s[:i], 10, 64)
		return id
	}
	return 0
}

func Tag(tag, format string, args ...any) {
	log.Printf("[%s gid=%d] "+format, append([]any{tag, GoID()}, args...)...)
}

func DumpGoroutines(reason string) {
	size := 1 << 20
	for {
		buf := make([]byte, size)
		n := runtime.Stack(buf, true)
		if n < size || size >= 8<<20 {
			log.Printf("[FZ-DUMP] reason=%q bytes=%d === BEGIN ===\n%s\n[FZ-DUMP] === END ===",
				reason, n, string(buf[:n]))
			return
		}
		size *= 2
	}
}

func StartWatchdog(name string, recheckEvery, stuckAfter time.Duration, ping func() <-chan struct{}) {
	go func() {
		t := time.NewTicker(recheckEvery)
		defer t.Stop()
		for range t.C {
			done := ping()
			select {
			case <-done:
				// healthy
			case <-time.After(stuckAfter):
				DumpGoroutines("watchdog " + name + ": main loop did not respond within " + stuckAfter.String())
				Tag("FZ-WD", "watchdog %q dumped & disarming (restart lazyflow to re-arm)", name)
				return
			}
		}
	}()
	Tag("FZ-INIT", "watchdog %q armed (recheck=%v stuckAfter=%v)", name, recheckEvery, stuckAfter)
}

func StartRuntimeSampler(interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		var ms runtime.MemStats
		for range t.C {
			runtime.ReadMemStats(&ms)
			log.Printf("[FZ-rt] goroutines=%d heapMB=%d nextGCMB=%d numGC=%d",
				runtime.NumGoroutine(),
				ms.HeapAlloc>>20,
				ms.NextGC>>20,
				ms.NumGC)
		}
	}()
}

func Setup() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for s := range sig {
			DumpGoroutines("signal=" + s.String())
		}
	}()
	log.Printf("[FZ-INIT gid=%d] SIGUSR1 → goroutine dump installed (kill -USR1 %d)",
		GoID(), os.Getpid())

	if os.Getenv("LAZYFLOW_PPROF") == "1" {
		addr := os.Getenv("LAZYFLOW_PPROF_ADDR")
		if addr == "" {
			addr = "127.0.0.1:6060"
		}
		go func() {
			log.Printf("[FZ-INIT gid=%d] pprof listening on http://%s/debug/pprof/", GoID(), addr)
			srv := &http.Server{
				Addr:              addr,
				ReadHeaderTimeout: 5 * time.Second,
			}
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("[FZ-INIT] pprof server stopped: %v", err)
			}
		}()
	}
}
