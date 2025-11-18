package cluster

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// LearnerStatus tracks the catch-up status of a learner node
type LearnerStatus struct {
	NodeID            string    // ID of the learner node
	MatchIndex        uint64    // Last log index replicated to learner
	CatchUpThreshold  uint64    // Index learner needs to reach to be caught up
	CaughtUp          bool      // Whether the learner is caught up
	LastContact       time.Time // Last time leader contacted learner
	AddedAt           time.Time // When the learner was added
	BytesReplicated   uint64    // Total bytes replicated (for progress tracking)
}

// MembershipManager handles cluster membership operations
type MembershipManager struct {
	mu                sync.RWMutex
	configManager     *ConfigurationManager
	learnerStatuses   map[string]*LearnerStatus // Track learner catch-up progress
	catchUpIndexGap   uint64                    // Max index gap for a learner to be considered caught up (default: 100)
	promotionCallback func(nodeID string) error // Callback when learner is ready for promotion
}

// NewMembershipManager creates a new membership manager
func NewMembershipManager(initialMembers map[string]string, catchUpIndexGap uint64) *MembershipManager {
	if catchUpIndexGap == 0 {
		catchUpIndexGap = 100 // Default: learner must be within 100 entries of leader
	}

	return &MembershipManager{
		configManager:   NewConfigurationManager(initialMembers),
		learnerStatuses: make(map[string]*LearnerStatus),
		catchUpIndexGap: catchUpIndexGap,
	}
}

// GetConfiguration returns the current cluster configuration
func (mm *MembershipManager) GetConfiguration() *Configuration {
	return mm.configManager.GetConfiguration()
}

// AddLearner proposes adding a new learner node to the cluster
func (mm *MembershipManager) AddLearner(nodeID, address string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  nodeID,
		Address: address,
		Index:   index,
		Term:    term,
	}

	if err := mm.configManager.ProposeChange(change); err != nil {
		return fmt.Errorf("failed to propose add learner: %w", err)
	}

	// Initialize learner status
	mm.learnerStatuses[nodeID] = &LearnerStatus{
		NodeID:           nodeID,
		MatchIndex:       0,
		CatchUpThreshold: 0, // Will be set when we know leader's last index
		CaughtUp:         false,
		LastContact:      time.Now(),
		AddedAt:          time.Now(),
		BytesReplicated:  0,
	}

	return nil
}

// CommitAddLearner commits a learner addition
func (mm *MembershipManager) CommitAddLearner(nodeID string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if err := mm.configManager.CommitChange(index, term); err != nil {
		return fmt.Errorf("failed to commit add learner: %w", err)
	}

	return nil
}

// UpdateLearnerProgress updates the replication progress for a learner
func (mm *MembershipManager) UpdateLearnerProgress(nodeID string, matchIndex, leaderLastIndex uint64, bytesReplicated uint64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	status, exists := mm.learnerStatuses[nodeID]
	if !exists {
		return
	}

	status.MatchIndex = matchIndex
	status.LastContact = time.Now()
	status.BytesReplicated += bytesReplicated

	// Update catch-up threshold if not set
	if status.CatchUpThreshold == 0 {
		status.CatchUpThreshold = leaderLastIndex
	}

	// Check if caught up
	indexGap := leaderLastIndex - matchIndex
	wasCaughtUp := status.CaughtUp
	status.CaughtUp = indexGap <= mm.catchUpIndexGap

	// Call promotion callback if learner just caught up
	if !wasCaughtUp && status.CaughtUp && mm.promotionCallback != nil {
		go mm.promotionCallback(nodeID)
	}
}

// IsLearnerCaughtUp checks if a learner has caught up with the leader
func (mm *MembershipManager) IsLearnerCaughtUp(nodeID string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	status, exists := mm.learnerStatuses[nodeID]
	if !exists {
		return false
	}

	return status.CaughtUp
}

// GetLearnerStatus returns the status of a learner
func (mm *MembershipManager) GetLearnerStatus(nodeID string) *LearnerStatus {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	status, exists := mm.learnerStatuses[nodeID]
	if !exists {
		return nil
	}

	// Return a copy
	return &LearnerStatus{
		NodeID:           status.NodeID,
		MatchIndex:       status.MatchIndex,
		CatchUpThreshold: status.CatchUpThreshold,
		CaughtUp:         status.CaughtUp,
		LastContact:      status.LastContact,
		AddedAt:          status.AddedAt,
		BytesReplicated:  status.BytesReplicated,
	}
}

// PromoteLearner proposes promoting a learner to a voting member
func (mm *MembershipManager) PromoteLearner(nodeID string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Check if learner is caught up
	status, exists := mm.learnerStatuses[nodeID]
	if !exists {
		return fmt.Errorf("learner %s not found", nodeID)
	}

	if !status.CaughtUp {
		return fmt.Errorf("learner %s is not caught up yet (matchIndex: %d, threshold: %d)",
			nodeID, status.MatchIndex, status.CatchUpThreshold)
	}

	change := &ConfigChange{
		Type:   PromoteNode,
		NodeID: nodeID,
		Index:  index,
		Term:   term,
	}

	if err := mm.configManager.ProposeChange(change); err != nil {
		return fmt.Errorf("failed to propose promote learner: %w", err)
	}

	return nil
}

// CommitPromoteLearner commits a learner promotion
func (mm *MembershipManager) CommitPromoteLearner(nodeID string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if err := mm.configManager.CommitChange(index, term); err != nil {
		return fmt.Errorf("failed to commit promote learner: %w", err)
	}

	// Remove from learner statuses
	delete(mm.learnerStatuses, nodeID)

	return nil
}

// RemoveNode proposes removing a node from the cluster
func (mm *MembershipManager) RemoveNode(nodeID string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	change := &ConfigChange{
		Type:   RemoveNode,
		NodeID: nodeID,
		Index:  index,
		Term:   term,
	}

	if err := mm.configManager.ProposeChange(change); err != nil {
		return fmt.Errorf("failed to propose remove node: %w", err)
	}

	return nil
}

// CommitRemoveNode commits a node removal
func (mm *MembershipManager) CommitRemoveNode(nodeID string, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if err := mm.configManager.CommitChange(index, term); err != nil {
		return fmt.Errorf("failed to commit remove node: %w", err)
	}

	// Remove from learner statuses if it was a learner
	delete(mm.learnerStatuses, nodeID)

	return nil
}

// IsChangeInProgress returns whether a configuration change is in progress
func (mm *MembershipManager) IsChangeInProgress() bool {
	return mm.configManager.IsChangeInProgress()
}

// GetPendingChange returns the pending configuration change if any
func (mm *MembershipManager) GetPendingChange() *ConfigChange {
	return mm.configManager.GetPendingChange()
}

// AbortChange aborts any pending configuration change
func (mm *MembershipManager) AbortChange() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	pending := mm.configManager.GetPendingChange()
	if pending != nil && pending.Type == AddLearnerNode {
		// Remove learner status if we're aborting a learner addition
		delete(mm.learnerStatuses, pending.NodeID)
	}

	mm.configManager.AbortChange()
}

// SetPromotionCallback sets a callback to be called when a learner is ready for promotion
func (mm *MembershipManager) SetPromotionCallback(callback func(nodeID string) error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	mm.promotionCallback = callback
}

// ApplyConfigChange applies a configuration change (used during log replay)
func (mm *MembershipManager) ApplyConfigChange(change *ConfigChange, index, term uint64) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if err := mm.configManager.ApplyConfigChange(change, index, term); err != nil {
		return err
	}

	// Update learner statuses
	switch change.Type {
	case AddLearnerNode:
		mm.learnerStatuses[change.NodeID] = &LearnerStatus{
			NodeID:           change.NodeID,
			MatchIndex:       0,
			CatchUpThreshold: 0,
			CaughtUp:         false,
			LastContact:      time.Now(),
			AddedAt:          time.Now(),
			BytesReplicated:  0,
		}

	case PromoteNode:
		delete(mm.learnerStatuses, change.NodeID)

	case RemoveNode:
		delete(mm.learnerStatuses, change.NodeID)
	}

	return nil
}

// SetConfiguration sets the configuration (used during snapshot restoration)
func (mm *MembershipManager) SetConfiguration(config *Configuration) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.configManager.SetConfiguration(config)

	// Reset learner statuses and rebuild from config
	mm.learnerStatuses = make(map[string]*LearnerStatus)
	for id, member := range config.Members {
		if member.Type == Learner {
			mm.learnerStatuses[id] = &LearnerStatus{
				NodeID:           id,
				MatchIndex:       0,
				CatchUpThreshold: 0,
				CaughtUp:         false,
				LastContact:      time.Now(),
				AddedAt:          member.AddedAt,
				BytesReplicated:  0,
			}
		}
	}
}

// GetAllLearnerStatuses returns status for all learners
func (mm *MembershipManager) GetAllLearnerStatuses() map[string]*LearnerStatus {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	statuses := make(map[string]*LearnerStatus)
	for id, status := range mm.learnerStatuses {
		statuses[id] = &LearnerStatus{
			NodeID:           status.NodeID,
			MatchIndex:       status.MatchIndex,
			CatchUpThreshold: status.CatchUpThreshold,
			CaughtUp:         status.CaughtUp,
			LastContact:      status.LastContact,
			AddedAt:          status.AddedAt,
			BytesReplicated:  status.BytesReplicated,
		}
	}
	return statuses
}

// Serialize serializes the configuration to JSON (for snapshot)
func (mm *MembershipManager) Serialize() ([]byte, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	config := mm.configManager.GetConfiguration()
	return json.Marshal(config)
}

// Deserialize deserializes the configuration from JSON (for snapshot restoration)
func (mm *MembershipManager) Deserialize(data []byte) error {
	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	mm.SetConfiguration(&config)
	return nil
}
