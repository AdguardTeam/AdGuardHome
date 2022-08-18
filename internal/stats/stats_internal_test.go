package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Use more realistic data.
func TestStatsCollector(t *testing.T) {
	ng := func(_ *unitDB) uint64 { return 0 }
	units := make([]*unitDB, 720)

	t.Run("hours", func(t *testing.T) {
		statsData := statsCollector(units, 0, Hours, ng)
		assert.Len(t, statsData, 720)
	})

	t.Run("days", func(t *testing.T) {
		for i := 0; i != 25; i++ {
			statsData := statsCollector(units, uint32(i), Days, ng)
			require.Lenf(t, statsData, 30, "i=%d", i)
		}
	})
}
