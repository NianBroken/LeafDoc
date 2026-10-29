// Package serviceports supplies the shared fixed listener allocation.
package serviceports

import (
	_ "embed"
	"encoding/json"
	"strconv"
)

//go:embed ports.json
var allocation []byte

// Port reads the authoritative allocation shared with the frontend and launcher.
func Port(service string) string {
	var ports map[string]int
	if err := json.Unmarshal(allocation, &ports); err != nil {
		panic(err)
	}
	port, ok := ports[service]
	if !ok {
		panic("unknown service port: " + service)
	}
	return strconv.Itoa(port)
}
