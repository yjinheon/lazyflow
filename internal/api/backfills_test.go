package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewClient(ClientConfig{BaseURL: srv.URL, Token: "test"})
	return c, srv
}

func TestListBackfills_ok(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v2/backfills") {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("dag_id"); got != "etl_daily" {
			t.Fatalf("dag_id query=%q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"backfills":     []map[string]any{{"id": 1, "dag_id": "etl_daily"}},
			"total_entries": 1,
		})
	}))
	defer srv.Close()

	got, err := c.ListBackfills(context.Background(), "etl_daily", nil)
	if err != nil {
		t.Fatalf("ListBackfills: %v", err)
	}
	if len(got.Backfills) != 1 || got.Backfills[0].ID != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestCancelBackfill_ok(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/backfills/42") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	if err := c.CancelBackfill(context.Background(), 42); err != nil {
		t.Fatalf("CancelBackfill: %v", err)
	}
}

func TestPauseUnpauseBackfill(t *testing.T) {
	calls := []map[string]any{}
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method=%s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		calls = append(calls, body)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7})
	}))
	defer srv.Close()
	if err := c.PauseBackfill(context.Background(), 7); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if err := c.UnpauseBackfill(context.Background(), 7); err != nil {
		t.Fatalf("Unpause: %v", err)
	}
	if len(calls) != 2 || calls[0]["is_paused"] != true || calls[1]["is_paused"] != false {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestDryRunBackfill_ok(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"logical_dates": []string{"2026-05-01", "2026-05-02"},
		})
	}))
	defer srv.Close()
	got, err := c.DryRunBackfill(context.Background(), map[string]any{"dag_id": "x"})
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}
	if len(got.LogicalDates) != 2 {
		t.Fatalf("dates=%v", got.LogicalDates)
	}
}

func TestListBackfills_err(t *testing.T) {
	c, srv := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"nope"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := c.ListBackfills(context.Background(), "x", nil); err == nil {
		t.Fatal("expected error")
	}
}
