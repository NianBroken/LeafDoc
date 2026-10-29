package tree

// checkNodeVersion enforces optimistic locking inside a write lock.
// Legacy nodes with no UpdatedAt (empty version) skip the check.
// Pass VersionUnchecked to bypass for internal system operations.
func checkNodeVersion(node *PageNode, expectedVersion string) error {
	if expectedVersion == VersionUnchecked {
		return nil
	}
	nodeVersion := node.Version()
	if nodeVersion == "" {
		return nil
	}
	if expectedVersion == "" {
		return ErrVersionRequired
	}
	if nodeVersion != expectedVersion {
		return ErrVersionConflict
	}
	return nil
}

func GeneratePathFromPageNode(entry *PageNode) string {
	path := ""
	if entry.Parent != nil {
		path = GeneratePathFromPageNode(entry.Parent) + "/" + entry.Slug
	} else {
		path = entry.Slug
	}
	return path
}
