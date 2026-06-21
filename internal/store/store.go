package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
)

type Query struct {
	Text   string
	Level  string
	Labels map[string]string
	Since  time.Time
	Limit  int
}

type Stats struct {
	Entries       int       `json:"entries"`
	IngestedTotal uint64    `json:"ingested_total"`
	LastIngest    time.Time `json:"last_ingest,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	path     string
	entries  []model.Entry
	byID     map[string]model.Entry
	index    map[string]map[string]struct{}
	seq      uint64
	ingested uint64
	last     time.Time
}

func New(path string) (*Store, error) {
	s := &Store{
		path:  path,
		byID:  make(map[string]model.Entry),
		index: make(map[string]map[string]struct{}),
	}
	if path == "" {
		return s, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Add(entry model.Entry) (model.Entry, error) {
	entry.Normalize(time.Now())
	if entry.ID == "" {
		entry.ID = s.nextID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)
	s.byID[entry.ID] = entry
	s.indexEntry(entry)
	atomic.AddUint64(&s.ingested, 1)
	s.last = entry.Timestamp

	if s.path != "" {
		if err := appendJSONL(s.path, entry); err != nil {
			return model.Entry{}, err
		}
	}

	return entry, nil
}

func (s *Store) Search(q Query) []model.Entry {
	if q.Limit <= 0 || q.Limit > 1000 {
		q.Limit = 100
	}

	terms := tokenize(q.Text)
	q.Level = strings.ToLower(strings.TrimSpace(q.Level))

	s.mu.RLock()
	defer s.mu.RUnlock()

	candidates := s.candidates(terms)
	out := make([]model.Entry, 0, min(q.Limit, len(candidates)))
	for _, entry := range candidates {
		if q.Level != "" && entry.Level != q.Level {
			continue
		}
		if !q.Since.IsZero() && entry.Timestamp.Before(q.Since) {
			continue
		}
		if !labelsMatch(entry.Labels, q.Labels) {
			continue
		}
		out = append(out, entry)
		if len(out) >= q.Limit {
			break
		}
	}
	return out
}

func (s *Store) Retain(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, nil
	}

	cutoff := time.Now().UTC().Add(-maxAge)
	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.entries[:0]
	removed := 0
	for _, entry := range s.entries {
		if entry.Timestamp.Before(cutoff) {
			removed++
			continue
		}
		kept = append(kept, entry)
	}
	s.entries = kept
	s.rebuildLocked()

	if removed > 0 && s.path != "" {
		if err := rewriteJSONL(s.path, s.entries); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		Entries:       len(s.entries),
		IngestedTotal: atomic.LoadUint64(&s.ingested),
		LastIngest:    s.last,
	}
}

func (s *Store) load() error {
	file, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry model.Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return fmt.Errorf("parse %s: %w", s.path, err)
		}
		entry.Normalize(time.Now())
		s.entries = append(s.entries, entry)
		s.byID[entry.ID] = entry
		s.indexEntry(entry)
	}
	return scanner.Err()
}

func (s *Store) nextID() string {
	next := atomic.AddUint64(&s.seq, 1)
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 36) + "-" + strconv.FormatUint(next, 36)
}

func (s *Store) indexEntry(entry model.Entry) {
	for _, token := range entryTokens(entry) {
		if s.index[token] == nil {
			s.index[token] = make(map[string]struct{})
		}
		s.index[token][entry.ID] = struct{}{}
	}
}

func (s *Store) rebuildLocked() {
	s.byID = make(map[string]model.Entry, len(s.entries))
	s.index = make(map[string]map[string]struct{})
	for _, entry := range s.entries {
		s.byID[entry.ID] = entry
		s.indexEntry(entry)
	}
}

func (s *Store) candidates(terms []string) []model.Entry {
	if len(terms) == 0 {
		out := make([]model.Entry, len(s.entries))
		copy(out, s.entries)
		sortEntries(out)
		return out
	}

	var ids map[string]struct{}
	for i, term := range terms {
		matches := s.index[term]
		if len(matches) == 0 {
			return nil
		}
		if i == 0 {
			ids = cloneSet(matches)
			continue
		}
		for id := range ids {
			if _, ok := matches[id]; !ok {
				delete(ids, id)
			}
		}
	}

	out := make([]model.Entry, 0, len(ids))
	for id := range ids {
		out = append(out, s.byID[id])
	}
	sortEntries(out)
	return out
}

func labelsMatch(entryLabels, wanted map[string]string) bool {
	for key, value := range wanted {
		if entryLabels[key] != value {
			return false
		}
	}
	return true
}

func entryTokens(entry model.Entry) []string {
	var raw []string
	raw = append(raw, entry.Level, entry.Message, entry.Source)
	for key, value := range entry.Labels {
		raw = append(raw, key, value, key+"="+value)
	}
	for key, value := range entry.Fields {
		raw = append(raw, key, value)
	}
	return tokenize(strings.Join(raw, " "))
}

func tokenize(text string) []string {
	seen := map[string]struct{}{}
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.' || r == '=' || r == ':')
	})
	for _, field := range fields {
		field = strings.Trim(field, " .,:;")
		if field != "" {
			seen[field] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for token := range seen {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func appendJSONL(path string, entry model.Entry) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(entry)
}

func rewriteJSONL(path string, entries []model.Entry) error {
	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sortEntries(entries []model.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})
}

func cloneSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}
