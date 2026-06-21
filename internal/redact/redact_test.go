package redact

import (
	"strings"
	"testing"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
)

func TestEntryRedactsSecrets(t *testing.T) {
	entry, count := Entry(model.Entry{
		Message: "login failed password=hunter2 bearer abc.def",
		Labels:  map[string]string{"service": "api"},
		Fields:  map[string]string{"api_key": "plain-secret"},
	})
	if count < 2 {
		t.Fatalf("expected at least two redactions, got %d", count)
	}
	if strings.Contains(entry.Message, "hunter2") || strings.Contains(entry.Message, "abc.def") {
		t.Fatalf("message was not redacted: %q", entry.Message)
	}
	if entry.Fields["api_key"] != "[REDACTED]" {
		t.Fatalf("field was not redacted: %+v", entry.Fields)
	}
}
