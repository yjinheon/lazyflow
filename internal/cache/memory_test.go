package cache

import (
	"testing"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func TestMemory_putGet(t *testing.T) {
	c := NewMemory(50 * time.Millisecond)
	bfs := []models.Backfill{{ID: 1}, {ID: 2}}
	c.PutBackfills("etl", bfs)
	got, ok := c.GetBackfills("etl")
	if !ok || len(got) != 2 || got[0].ID != 1 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
}

func TestMemory_ttl(t *testing.T) {
	c := NewMemory(20 * time.Millisecond)
	c.PutBackfills("x", []models.Backfill{{ID: 9}})
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.GetBackfills("x"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestMemory_missForUnknownKey(t *testing.T) {
	c := NewMemory(time.Second)
	if _, ok := c.GetBackfills("nope"); ok {
		t.Fatal("expected miss")
	}
}

func TestMemory_close(t *testing.T) {
	c := NewMemory(time.Second)
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
