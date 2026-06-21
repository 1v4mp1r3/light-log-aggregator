package syslog

import (
	"testing"
	"time"
)

func TestParseRFC3164ishLine(t *testing.T) {
	entry := Parse("<13>Jun 21 10:11:12 host-a sshd[123]: Failed password for root", time.Date(2026, 6, 21, 10, 12, 0, 0, time.UTC))
	if entry.Labels["host"] != "host-a" || entry.Labels["app"] != "sshd" {
		t.Fatalf("labels not parsed: %+v", entry.Labels)
	}
	if entry.Message != "Failed password for root" {
		t.Fatalf("message not parsed: %q", entry.Message)
	}
}
