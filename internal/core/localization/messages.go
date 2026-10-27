// Package localization supplies the same error catalogue to API clients and mail senders.
package localization

import (
	"embed"
	"encoding/json"
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
		return template
	}
	for _, arg := range args {
		value = strings.Replace(value, "%s", arg, 1)
	}
	return value
}

// Fallback is used for unstructured failures, never for document content or identifiers.
func Fallback(status int) string {
	values := map[string][2]string{
		"zh": {"操作成功", "操作失败，请检查输入和访问权限后重试"},
		"en": {"Operation completed", "Operation failed. Check the input and access permissions."},
		"de": {"Vorgang abgeschlossen", "Vorgang fehlgeschlagen. Prüfen Sie Eingabe und Zugriffsrechte."},
		"es": {"Operación completada", "La operación falló. Revise los datos y los permisos de acceso."},
		"fr": {"Opération terminée", "Échec de l’opération. Vérifiez les données et les autorisations."},
	}
	value, ok := values[Language()]
	if !ok {
		value = values["zh"]
	}
	if status < 400 {
		return value[0]
	}
	return value[1]
}
