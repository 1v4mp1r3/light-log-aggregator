package model

import (
	"strings"
	"time"
)

type Entry struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	Source    string            `json:"source,omitempty"`
}

func (e *Entry) Normalize(now time.Time) {
	if e.Timestamp.IsZero() {
		e.Timestamp = now.UTC()
	} else {
		e.Timestamp = e.Timestamp.UTC()
	}

	e.Level = strings.ToLower(strings.TrimSpace(e.Level))
	if e.Level == "" {
		e.Level = "info"
	}

	e.Message = strings.TrimSpace(e.Message)
	if e.Labels == nil {
		e.Labels = map[string]string{}
	}
	if e.Fields == nil {
		e.Fields = map[string]string{}
	}
}
