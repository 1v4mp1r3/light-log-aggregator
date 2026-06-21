package syslog

import (
	"regexp"
	"strings"
	"time"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
)

var rfc3164ish = regexp.MustCompile(`^(?:<\d+>)?([A-Z][a-z]{2}\s+\d{1,2}\s+\d\d:\d\d:\d\d)\s+(\S+)\s+([^\[:\s]+)(?:\[\d+\])?:\s?(.*)$`)

func Parse(line string, now time.Time) model.Entry {
	line = strings.TrimSpace(line)
	entry := model.Entry{
		Timestamp: now.UTC(),
		Level:     "info",
		Message:   line,
		Source:    "syslog",
		Labels:    map[string]string{"transport": "udp-syslog"},
	}

	match := rfc3164ish.FindStringSubmatch(line)
	if len(match) != 5 {
		return entry
	}

	if ts, ok := parseTimestamp(match[1], now); ok {
		entry.Timestamp = ts
	}
	entry.Labels["host"] = match[2]
	entry.Labels["app"] = match[3]
	entry.Message = match[4]
	return entry
}

func parseTimestamp(raw string, now time.Time) (time.Time, bool) {
	value := now.Format("2006") + " " + raw
	ts, err := time.ParseInLocation("2006 Jan 2 15:04:05", value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return ts.UTC(), true
}
