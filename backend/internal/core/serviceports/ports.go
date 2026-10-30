// Package serviceports supplies the shared fixed listener allocation.
package serviceports

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

//go:embed ports.json
var allocation []byte

// Names returns a detached, stable list of allocated service names.
func Names() []string {
	var ports map[string]int
	if err := json.Unmarshal(allocation, &ports); err != nil {
		panic(err)
	}
	names := make([]string, 0, len(ports))
	for name := range ports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Port reads the authoritative allocation shared with the frontend and launcher.
func Port(service string) string {
	return strconv.Itoa(port(service))
}

func port(service string) int {
	var ports map[string]int
	if err := json.Unmarshal(allocation, &ports); err != nil {
		panic(err)
	}
	value, ok := ports[service]
	if !ok {
		panic("unknown service port: " + service)
	}
	return value
}

// Validator rejects overrides of a service's permanent listener allocation.
func Validator(service string) func(string) error {
	return func(value string) error {
		if value != Port(service) {
			return fmt.Errorf("%s must use fixed port %s", service, Port(service))
		}
		return nil
	}
}
