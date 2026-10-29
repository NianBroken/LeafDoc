package tags

import "github.com/NianBroken/LeafDoc/internal/core/excerpt"

// ExtractExcerptFromContent parses markdown (including frontmatter) and returns a short plain-text excerpt.
func ExtractExcerptFromContent(raw string) string {
	return excerpt.FromContent(raw)
}
