// +build ignore

package dnsfilter

import (
	"fmt"
	"sort"
	"testing"
)

// This is a simple tool that takes a list of services and prints them to the output.
// It is supposed to be used to update:
// client/src/helpers/constants.js
// client/src/components/ui/Icons.js
//
// Usage:
// 1. go run ./internal/dnsfilter/blocked_test.go
// 2. Use the output to replace `SERVICES` array in "client/src/helpers/constants.js".
// 3. You'll need to enter services names manually.
// 4. Don't forget to add missing icons to "client/src/components/ui/Icons.js".
//
// TODO(ameshkov): Rework generator: have a JSON file with all the metadata we need
// then use this JSON file to generate JS and Go code
func TestGenServicesArray(t *testing.T) {
	services := make([]svc, len(serviceRulesArray))
	copy(services, serviceRulesArray)

	sort.Slice(services, func(i, j int) bool {
		return services[i].name < services[j].name
	})

	fmt.Println("export const SERVICES = [")
	for _, s := range services {
		fmt.Printf("    {\n        id: '%s',\n        name: '%s',\n    },\n", s.name, s.name)
	}
	fmt.Println("];")
}
