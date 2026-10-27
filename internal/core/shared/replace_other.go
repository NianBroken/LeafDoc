//go:build !windows

package shared

import "os"

// atomicReplace commits a file using the filesystem replacement operation.
func atomicReplace(src, dst string) error { return os.Rename(src, dst) }
