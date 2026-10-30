package tree

import (
	"os"
	"sort"
	"strings"
)

// sortReconstructionEntries provides stable filename ordering for import sources.
func sortReconstructionEntries(entries []os.DirEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i].Name(), entries[j].Name()
		la, lb := strings.ToLower(a), strings.ToLower(b)
		if la == lb {
			return a < b
		}
		return la < lb
	})
}
