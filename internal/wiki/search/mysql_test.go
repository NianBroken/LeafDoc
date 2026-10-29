package search

import (
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"testing"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "internal/wiki/search") }
