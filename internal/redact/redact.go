package redact

import (
	"regexp"
	"strings"

	"github.com/1v4mp1r3/light-log-aggregator/internal/model"
)

type pattern struct {
	repl string
	expr *regexp.Regexp
}

var patterns = []pattern{
	{expr: regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|api[_-]?key|secret)=([^\s,&]+)`), repl: `$1=[REDACTED]`},
	{expr: regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`), repl: `bearer [REDACTED]`},
	{expr: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), repl: `[AWS_ACCESS_KEY_REDACTED]`},
	{expr: regexp.MustCompile(`(?i)\b(authorization|x-api-key):\s*[^\s]+`), repl: `$1: [REDACTED]`},
}

func Entry(in model.Entry) (model.Entry, int) {
	var count int
	in.Message, count = String(in.Message, count)

	for key, value := range in.Labels {
		if suspiciousKey(key) {
			in.Labels[key] = "[REDACTED]"
			count++
			continue
		}
		in.Labels[key], count = String(value, count)
	}

	for key, value := range in.Fields {
		if suspiciousKey(key) {
			in.Fields[key] = "[REDACTED]"
			count++
			continue
		}
		in.Fields[key], count = String(value, count)
	}

	return in, count
}

func String(value string, count int) (string, int) {
	out := value
	for _, p := range patterns {
		next := p.expr.ReplaceAllString(out, p.repl)
		if next != out {
			count++
			out = next
		}
	}
	return out, count
}

func suspiciousKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "api_key") ||
		strings.Contains(key, "apikey") ||
		strings.Contains(key, "authorization")
}
