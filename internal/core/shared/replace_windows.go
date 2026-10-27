package shared

import "golang.org/x/sys/windows"

// atomicReplace commits a file without deleting the destination before replacement.
func atomicReplace(src, dst string) error {
	source, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	destination, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, destination, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
