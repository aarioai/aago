package types

import (
	"log"
	"regexp"
	"strings"
	"time"
)

// ParseDuration parses a duration string.
// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
// A duration string is a possibly signed sequence of decimal numbers,
// each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0ns" || s == "0us" || s == "0µs" || s == "0ms" || s == "0s" || s == "0m" || s == "0h" {
		return 0, nil
	}
	return time.ParseDuration(s)
}

func ParseDurationWithDefault(s string, defaultValue time.Duration) time.Duration {
	d, err := ParseDuration(s)
	if err != nil {
		log.Println("[error] invalid time duration: " + s)
		return defaultValue
	}
	return d
}

// StringifyDuration stringifies time duration to a duration string and trim end zeroes.
func StringifyDuration(d time.Duration) string {
	s := d.String()
	if !strings.Contains(s, "0") {
		return s
	}
	var durationPartRe = regexp.MustCompile(`([0-9.]+)(ns|µs|us|ms|s|m|h)`)
	parts := durationPartRe.FindAllStringSubmatch(s, -1)
	for len(parts) > 1 && parts[len(parts)-1][1] == "0" {
		parts = parts[:len(parts)-1]
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p[0])
	}
	return b.String()
}
