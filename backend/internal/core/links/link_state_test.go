package links

import (
	"testing"
)

func TestClassifyTitleMatches(t *testing.T) {
	cases := []struct {
		matches int
		want    LinkState
	}{
		{0, LinkBroken},
		{1, LinkResolved},
		{2, LinkAmbiguous},
		{5, LinkAmbiguous},
	}
	for _, tc := range cases {
		if got := classifyTitleMatches(tc.matches); got != tc.want {
			t.Errorf("classifyTitleMatches(%d) = %v, want %v", tc.matches, got, tc.want)
		}
	}
}
