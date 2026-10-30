// Command service-ports exposes the shared listener policy to source-tree tools.
package main

import (
	"fmt"
	"os"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
)

func main() {
	if err := serviceports.Release(os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
