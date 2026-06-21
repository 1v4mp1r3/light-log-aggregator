package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
)

func TestSearchByTextAndLabels(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _ = s.Add(model.Entry{Message: "api request failed", Level: "error", Labels: map[string]string{"service": "api"}})
	_, _ = s.Add(model.Entry{Message: "worker heartbeat", Level: "info", Labels: map[string]string{"service": "worker"}})

	got := s.Search(Query{
		Text:   "failed",
		Level:  "error",
		Labels: map[string]string{"service": "api"},
		Limit:  10,
	})
	if len(got) != 1 || got[0].Message != "api request failed" {
		t.Fatalf("unexpected search result: %+v", got)
	}
}

func TestStorePersistsJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := s.Add(model.Entry{Message: "persisted", Labels: map[string]string{"service": "api"}}); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	loaded, err := New(path)
	if err != nil {
		t.Fatalf("New load returned error: %v", err)
	}
	got := loaded.Search(Query{Text: "persisted", Limit: 10})
	if len(got) != 1 {
		t.Fatalf("expected persisted entry, got %+v", got)
	}
}

func TestRetentionRemovesOldEntries(t *testing.T) {
	s, err := New("")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	_, _ = s.Add(model.Entry{Timestamp: time.Now().Add(-48 * time.Hour), Message: "old"})
	_, _ = s.Add(model.Entry{Timestamp: time.Now(), Message: "new"})

	removed, err := s.Retain(24 * time.Hour)
	if err != nil {
		t.Fatalf("Retain returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected one removed entry, got %d", removed)
	}
	if stats := s.Stats(); stats.Entries != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
