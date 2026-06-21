package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1v4mp1r3/light-log-aggregator/internal/store"
)

func TestHTTPIngestSearchAndRedact(t *testing.T) {
	s, err := store.New("")
	if err != nil {
		t.Fatalf("New store returned error: %v", err)
	}
	handler := New(s, "test").Routes()

	body := bytes.NewBufferString(`{"level":"error","message":"failed password=hunter2","labels":{"service":"api"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/logs", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected ingest status %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/search?q=failed&label=service=api", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected search status %d", rec.Code)
	}

	var payload struct {
		Count   int `json:"count"`
		Results []struct {
			Message string `json:"message"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}
	if payload.Count != 1 {
		t.Fatalf("expected one result, got %+v", payload)
	}
	if payload.Results[0].Message == "failed password=hunter2" {
		t.Fatal("secret was not redacted")
	}
}
