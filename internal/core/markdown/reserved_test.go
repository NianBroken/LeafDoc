package markdown

import "testing"

func TestIsSystemKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"tags", true},
		{"Tags", true},
		{"TAGS", true},
		{"leafdoc_id", true},
		{"leafdoc_title", true},
		{"leafdoc_created_at", true},
		{"LEAFDOC_ANYTHING", true},
		// title is NOT a system key — it is a user-defined custom property
		// that may coexist alongside leafdoc_title.
		{"title", false},
		{"Title", false},
		{"status", false},
		{"author", false},
		{"permalink", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			got := IsSystemKey(tc.key)
			if got != tc.want {
				t.Fatalf("IsSystemKey(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}
