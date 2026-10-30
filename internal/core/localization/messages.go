// Package localization supplies the same error catalogue to API clients and mail senders.
package localization

import (
	"embed"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
)

//go:embed *.json
var files embed.FS

//go:generate node ../../../scripts/sync-error-catalogs.mjs
var mu sync.RWMutex
var language = func() string { return "zh" }

// SetLanguageProvider reads committed instance preferences at the moment a message is generated.
func SetLanguageProvider(provider func() string) { mu.Lock(); language = provider; mu.Unlock() }
func Language() string                           { mu.RLock(); fn := language; mu.RUnlock(); return fn() }

var catalogues = load()

// Numeric validation arguments share a catalogue template rather than fixed limits.
var numbers = regexp.MustCompile(`[0-9]+`)

func load() map[string]map[string]string {
	result := map[string]map[string]string{}
	for _, lang := range []string{"zh", "en", "de", "es", "fr"} {
		data, err := files.ReadFile(lang + ".json")
		if err != nil {
			panic(err)
		}
		var v map[string]string
		if err = json.Unmarshal(data, &v); err != nil {
			panic(err)
		}
		result[lang] = v
	}
	return result
}

// Message expands stable catalogue templates and leaves user-provided arguments unchanged.
func Message(template string, args ...string) string {
	value := catalogues[Language()][template]
	if value == "" {
		value = catalogues[Language()][strings.ToLower(template)]
	}
	if value == "" && len(args) == 0 {
		key := numbers.ReplaceAllString(strings.ToLower(template), "%s")
		value = catalogues[Language()][key]
		if value != "" {
			args = numbers.FindAllString(template, -1)
		}
	}
	if value == "" {
		return template
	}
	for _, arg := range args {
		value = strings.Replace(value, "%s", arg, 1)
	}
	return value
}

// Fallback is used for unstructured failures, never for document content or identifiers.
func Fallback(status int) string {
	key := "request failed"
	switch {
	case status < 400:
		key = "operation completed"
	case status == 400 || status == 422:
		key = "invalid request input"
	case status == 401:
		key = "authentication required"
	case status == 403:
		key = "permission denied"
	case status == 404:
		key = "resource not found"
	case status == 409:
		key = "resource conflict"
	case status == 413:
		key = "uploaded file too large"
	case status == 429:
		key = "too many requests"
	case status >= 500:
		key = "service request failed"
	}
	return Message(key)
}
