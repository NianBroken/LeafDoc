package health

import (
	"testing"

	"github.com/NianBroken/LeafDoc/internal/testutil"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "health") }
