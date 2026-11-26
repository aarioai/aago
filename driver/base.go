package driver

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aarioai/aago/aa"
	"github.com/aarioai/aago/aa/ae"
	"github.com/aarioai/aago/pkg/afmt"
	"github.com/aarioai/aago/pkg/types"
)

func allEmpty(values ...string) bool {
	for _, v := range values {
		if v != "" {
			return false
		}
	}
	return true
}

// parseTimeouts parses connection, read, write timeout
// 10s, 1000ms
func parseTimeouts(t string, defaultTimeouts ...time.Duration) (conn time.Duration, read time.Duration, write time.Duration) {
	for i, t := range defaultTimeouts {
		switch i {
		case 0:
			conn = t
		case 1:
			read = t
		case 2:
			write = t
		}
	}

	ts := strings.Split(strings.Replace(t, " ", "", -1), ",")
	for i, t := range ts {
		switch i {
		case 0:
			if conn2, err := types.ParseDuration(t); err == nil && conn2 > 0 {
				conn = conn2
			}
		case 1:
			if read2, err := types.ParseDuration(t); err == nil && read2 > 0 {
				read = read2
			}
		case 2:
			if write2, err := types.ParseDuration(t); err == nil && write2 > 0 {
				write = write2
			}
		}
	}
	return
}

// tryGetSectionCfg reads driver config keys for a named section.
//
// Preferred YAML layout (aago v1.0.0 aconfig DEFAULT inheritance):
//
//	mysql:
//	  DEFAULT: {host: ...}
//	  test: {}
//	  hello: {schema: helloworld}
//
// Lookup order for base="mysql", section="test", key="host":
//  1. mysql.test.host
//  2. mysql.DEFAULT.host
//  3. legacy flat keys: test.host, mysql_test.host, mysql.host
func tryGetSectionCfg(app *aa.App, base, section string, key string, defaultValue ...string) (string, error) {
	defaultV := afmt.First(defaultValue)

	raw := section
	if raw == "" || raw == base {
		raw = "DEFAULT"
	}
	sec := trimSectionPrefix(base, raw)

	prefixes := []string{
		base + "." + sec,
	}
	if sec != "DEFAULT" {
		prefixes = append(prefixes, base+".DEFAULT")
	}
	// Legacy flat INI-style keys (pre-v1.0.0)
	if sec == "DEFAULT" {
		prefixes = append(prefixes, base)
	} else {
		prefixes = append(prefixes, sec, base+"_"+sec, base)
	}

	var lastErr error
	for _, prefix := range prefixes {
		for _, k := range keyVariants(key) {
			v, err := app.Config.MustGetString(prefix + "." + k)
			if err == nil {
				return v, nil
			}
			lastErr = err
		}
	}
	if defaultV != "" {
		return defaultV, nil
	}
	return "", lastErr
}

// trimSectionPrefix strips the driver base name (and its aliases) from a
// section name, so `mysql_test` and `postgres_test` normalize to `test`.
func trimSectionPrefix(base, section string) string {
	for _, p := range sectionPrefixes(base) {
		if strings.HasPrefix(section, p) {
			return strings.TrimPrefix(section, p)
		}
	}
	return section
}

func sectionPrefixes(base string) []string {
	prefixes := []string{base + "_"}
	if base == POSTGRESQL {
		prefixes = append(prefixes, "postgres_")
	}
	return prefixes
}

// keyVariants returns snake_case, camelCase and PascalCase forms of a config key.
func keyVariants(key string) []string {
	if !strings.Contains(key, "_") {
		return []string{key}
	}
	parts := strings.Split(key, "_")
	camel := parts[0]
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		camel += camelSegment(p)
	}
	title := strings.ToUpper(camel[:1]) + camel[1:]
	if camel == key || title == key {
		return []string{key}
	}
	return []string{key, camel, title}
}

// camelSegment capitalizes a snake_case segment, keeping well-known acronyms
// such as http2/url/tls uppercase to match existing camelCase config keys.
func camelSegment(s string) string {
	switch s {
	case "http", "http2", "https", "url", "uri", "tls", "api", "id", "db", "dsn", "ip", "ttl":
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func getSectionCfgWithFallback(app *aa.App, base, section string, keys ...string) string {
	v, _ := tryGetSectionCfgWithFallback(app, base, section, keys...)
	return v
}

func tryGetSectionCfgWithFallback(app *aa.App, base, section string, keys ...string) (string, error) {
	var lastErr error
	for _, k := range keys {
		v, err := tryGetSectionCfg(app, base, section, k)
		if err == nil {
			return v, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func getSectionCfgDurationWithFallback(app *aa.App, base, section string, keys ...string) (time.Duration, error) {
	s := getSectionCfgWithFallback(app, base, section, keys...)
	if s == "" {
		return 0, nil
	}
	return types.ParseDuration(s)
}

func getSectionCfgDurationWithDefault(app *aa.App, base, section string, fallback time.Duration, keys ...string) (time.Duration, error) {
	s := getSectionCfgWithFallback(app, base, section, keys...)
	if s == "" {
		return fallback, nil
	}
	return types.ParseDuration(s)
}

func getSectionCfgBoolWithFallback(app *aa.App, base, section string, keys ...string) (bool, error) {
	s := getSectionCfgWithFallback(app, base, section, keys...)
	if s == "" {
		return false, nil
	}
	return strconv.ParseBool(s)
}

func normalizePoolSection(base, section string) string {
	if section == "" || section == "DEFAULT" || section == base {
		return "DEFAULT"
	}
	return trimSectionPrefix(base, section)
}

func CloseAllPools(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		CloseInfluxdbPool()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		CloseMongodbPool(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		CloseMysqlPool()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ClosePostgresPool()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		CloseRabbitmqPool()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		CloseRedisPool()
	}()

	wg.Wait()
}

func newConfigError(base, section string, err error) *ae.Error {
	return ae.NewF(ae.VariantAlsoNegotiates, "config %s.%s error: %s", base, section, err.Error())
}

func parseStrings(s, separator string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, " ", "")
	return strings.Split(s, separator)
}
func parseUint16s(s, separator string) []uint16 {
	arr := parseStrings(s, separator)
	if len(arr) == 0 {
		return nil
	}
	ret := make([]uint16, len(arr))
	for i, v := range arr {
		ret[i] = types.ToUint16(v)
	}
	return ret
}
