package overseer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScoutStore(t *testing.T) {
	t.Run("NewScoutStore", func(t *testing.T) {
		store := NewScoutStore()
		assert.NotNil(t, store)
		assert.NotNil(t, store.scouts)
	})

	t.Run("UpdateScout", func(t *testing.T) {
		store := NewScoutStore()
		nodeID := "test-node"
		address := "127.0.0.1"

		// Test new scout creation
		store.UpdateScout(nodeID, address)
		scout, exists := store.scouts[nodeID]
		assert.True(t, exists)
		assert.Equal(t, nodeID, scout.NodeID)
		assert.Equal(t, address, scout.Address)
		assert.Equal(t, ScoutStatusActive, scout.Status)

		// Test scout update
		store.UpdateScout(nodeID, address)
		scout, exists = store.scouts[nodeID]
		assert.True(t, exists)
		assert.Equal(t, ScoutStatusActive, scout.Status)
	})

	t.Run("GetActiveScouts", func(t *testing.T) {
		store := NewScoutStore()

		// Add active and inactive scouts
		store.UpdateScout("active1", "127.0.0.1")
		store.UpdateScout("active2", "127.0.0.2")
		store.scouts["inactive"] = &Scout{
			NodeID:   "inactive",
			Status:   ScoutStatusInactive,
			LastSeen: time.Now(),
		}

		activeScouts := store.GetActiveScouts()
		assert.Len(t, activeScouts, 2)
		for _, scout := range activeScouts {
			assert.Equal(t, ScoutStatusActive, scout.Status)
		}
	})

	t.Run("GetAllScouts", func(t *testing.T) {
		store := NewScoutStore()

		// Add multiple scouts
		store.UpdateScout("node1", "127.0.0.1")
		store.UpdateScout("node2", "127.0.0.2")
		store.scouts["inactive"] = &Scout{
			NodeID:   "inactive",
			Status:   ScoutStatusInactive,
			LastSeen: time.Now(),
		}

		allScouts := store.GetAllScouts()
		assert.Len(t, allScouts, 3)
	})
}
