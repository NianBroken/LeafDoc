package properties

import (
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"testing"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "properties") }
