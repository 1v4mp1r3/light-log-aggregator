package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
	"github.com/1v4mp1r3/light-log-aggregator/internal/redact"
	"github.com/1v4mp1r3/light-log-aggregator/internal/store"
)

type Server struct {
	store      *store.Store
	version    string
	redactions uint64
}

func New(s *store.Store, version string) *Server {
	return &Server{store: s, version: version}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleUI)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/healthz", s.handleHealth)
	return withHeaders(mux)
}

func (s *Server) Add(entry model.Entry) (model.Entry, error) {
	entry, count := redact.Entry(entry)
	if count > 0 {
		atomic.AddUint64(&s.redactions, uint64(count))
	}
	return s.store.Add(entry)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	entries, err := decodeEntries(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := make([]model.Entry, 0, len(entries))
	for _, entry := range entries {
		saved, err := s.Add(entry)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, saved)
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": len(out), "entries": out})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := store.Query{
		Text:   r.URL.Query().Get("q"),
		Level:  r.URL.Query().Get("level"),
		Labels: parseLabelFilters(r.URL.Query()["label"]),
		Limit:  parseLimit(r.URL.Query().Get("limit")),
	}
	if since := r.URL.Query().Get("since"); since != "" {
		if d, err := time.ParseDuration(since); err == nil {
			query.Since = time.Now().UTC().Add(-d)
		}
	}

	results := s.store.Search(query)
	writeJSON(w, http.StatusOK, map[string]any{"count": len(results), "results": results})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	stats := s.store.Stats()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP loglite_entries Current retained log entries.\n")
	fmt.Fprintf(w, "# TYPE loglite_entries gauge\n")
	fmt.Fprintf(w, "loglite_entries %d\n", stats.Entries)
	fmt.Fprintf(w, "# HELP loglite_ingested_total Total accepted log entries since process start.\n")
	fmt.Fprintf(w, "# TYPE loglite_ingested_total counter\n")
	fmt.Fprintf(w, "loglite_ingested_total %d\n", stats.IngestedTotal)
	fmt.Fprintf(w, "# HELP loglite_redactions_total Total redaction operations since process start.\n")
	fmt.Fprintf(w, "# TYPE loglite_redactions_total counter\n")
	fmt.Fprintf(w, "loglite_redactions_total %d\n", atomic.LoadUint64(&s.redactions))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": s.version})
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(uiHTML))
}

func decodeEntries(body []byte) ([]model.Entry, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("empty request body")
	}

	var batch []model.Entry
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &batch); err != nil {
			return nil, err
		}
		return batch, nil
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) > 1 {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var entry model.Entry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				return nil, err
			}
			batch = append(batch, entry)
		}
		return batch, nil
	}

	var entry model.Entry
	if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
		return nil, err
	}
	return []model.Entry{entry}, nil
}

func parseLabelFilters(values []string) map[string]string {
	out := map[string]string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, val, ok := strings.Cut(part, "=")
			if !ok {
				key, val, ok = strings.Cut(part, ":")
			}
			if ok && strings.TrimSpace(key) != "" {
				out[strings.TrimSpace(key)] = strings.TrimSpace(val)
			}
		}
	}
	return out
}

func parseLimit(raw string) int {
	if raw == "" {
		return 100
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 100
	}
	return limit
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
