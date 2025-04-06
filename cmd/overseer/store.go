package overseer

import (
	"sync"
	"time"
)

// InactiveScoutFactor determines how many intervals a scout can be inactive before being considered unhealthy.
const InactiveScoutFactor = 2

const (
	// ScoutStatusActive indicates a scout is currently active.
	ScoutStatusActive = "active"
	// ScoutStatusInactive indicates a scout is currently inactive.
	ScoutStatusInactive = "inactive"
)

// Scout represents a network scout agent.
type Scout struct {
	NodeID   string    `json:"node_id"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"`
	Address  string    `json:"address"`
}

// ScoutStore manages the storage and retrieval of scout information.
type ScoutStore struct {
	scouts     map[string]*Scout
	scoutsLock sync.RWMutex
}

// NewScoutStore creates a new ScoutStore instance.
func NewScoutStore() *ScoutStore {
	return &ScoutStore{
		scouts: make(map[string]*Scout),
	}
}

// GetActiveScouts returns all currently active scouts.
func (s *ScoutStore) GetActiveScouts() []*Scout {
	s.scoutsLock.RLock()
	defer s.scoutsLock.RUnlock()

	scouts := make([]*Scout, 0, len(s.scouts))
	for _, scout := range s.scouts {
		if scout.Status == ScoutStatusActive {
			scouts = append(scouts, scout)
		}
	}
	return scouts
}

// GetAllScouts returns all registered scouts.
func (s *ScoutStore) GetAllScouts() []Scout {
	s.scoutsLock.RLock()
	defer s.scoutsLock.RUnlock()

	scouts := make([]Scout, 0, len(s.scouts))
	for _, scout := range s.scouts {
		scouts = append(scouts, *scout)
	}
	return scouts
}

// UpdateScout updates the information for a specific scout.
func (s *ScoutStore) UpdateScout(nodeID string, address string) {
	s.scoutsLock.Lock()
	defer s.scoutsLock.Unlock()

	scout, exists := s.scouts[nodeID]
	if !exists {
		s.scouts[nodeID] = &Scout{
			NodeID:   nodeID,
			LastSeen: time.Now(),
			Status:   ScoutStatusActive,
			Address:  address,
		}
		return
	}

	scout.LastSeen = time.Now()
	scout.Status = ScoutStatusActive
}

// CheckHealth periodically verifies the health status of all scouts.
func (s *ScoutStore) CheckHealth(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		s.scoutsLock.Lock()
		now := time.Now()
		for _, scout := range s.scouts {
			if now.Sub(scout.LastSeen) > (interval * InactiveScoutFactor) {
				scout.Status = ScoutStatusInactive
			}
		}
		s.scoutsLock.Unlock()
	}
}
