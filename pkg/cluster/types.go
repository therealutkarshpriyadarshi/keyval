package cluster

import (
	"fmt"
	"sync"
	"time"
)

// MemberType represents the type of cluster member
type MemberType int

const (
	// Voter is a full voting member of the cluster
	Voter MemberType = iota
	// Learner is a non-voting member that receives log entries but doesn't participate in quorum
	Learner
)

func (mt MemberType) String() string {
	switch mt {
	case Voter:
		return "Voter"
	case Learner:
		return "Learner"
	default:
		return "Unknown"
	}
}

// Member represents a node in the cluster
type Member struct {
	ID      string     // Unique identifier for the member
	Address string     // Network address (host:port)
	Type    MemberType // Voter or Learner
	AddedAt time.Time  // When the member was added
}

// Configuration represents the cluster configuration at a point in time
type Configuration struct {
	Members map[string]*Member // Map of member ID to Member
	Index   uint64             // Log index where this config was committed
	Term    uint64             // Term when this config was committed
}

// NewConfiguration creates a new cluster configuration
func NewConfiguration() *Configuration {
	return &Configuration{
		Members: make(map[string]*Member),
	}
}

// Clone creates a deep copy of the configuration
func (c *Configuration) Clone() *Configuration {
	clone := &Configuration{
		Members: make(map[string]*Member),
		Index:   c.Index,
		Term:    c.Term,
	}
	for id, member := range c.Members {
		clone.Members[id] = &Member{
			ID:      member.ID,
			Address: member.Address,
			Type:    member.Type,
			AddedAt: member.AddedAt,
		}
	}
	return clone
}

// AddMember adds a member to the configuration
func (c *Configuration) AddMember(id, address string, memberType MemberType) error {
	if _, exists := c.Members[id]; exists {
		return fmt.Errorf("member %s already exists in configuration", id)
	}
	c.Members[id] = &Member{
		ID:      id,
		Address: address,
		Type:    memberType,
		AddedAt: time.Now(),
	}
	return nil
}

// RemoveMember removes a member from the configuration
func (c *Configuration) RemoveMember(id string) error {
	if _, exists := c.Members[id]; !exists {
		return fmt.Errorf("member %s not found in configuration", id)
	}
	delete(c.Members, id)
	return nil
}

// PromoteLearner promotes a learner to a voting member
func (c *Configuration) PromoteLearner(id string) error {
	member, exists := c.Members[id]
	if !exists {
		return fmt.Errorf("member %s not found in configuration", id)
	}
	if member.Type != Learner {
		return fmt.Errorf("member %s is not a learner", id)
	}
	member.Type = Voter
	return nil
}

// GetVoters returns all voting members
func (c *Configuration) GetVoters() []*Member {
	voters := make([]*Member, 0)
	for _, member := range c.Members {
		if member.Type == Voter {
			voters = append(voters, member)
		}
	}
	return voters
}

// GetLearners returns all learner members
func (c *Configuration) GetLearners() []*Member {
	learners := make([]*Member, 0)
	for _, member := range c.Members {
		if member.Type == Learner {
			learners = append(learners, member)
		}
	}
	return learners
}

// VoterCount returns the number of voting members
func (c *Configuration) VoterCount() int {
	count := 0
	for _, member := range c.Members {
		if member.Type == Voter {
			count++
		}
	}
	return count
}

// Quorum returns the quorum size (majority of voters)
func (c *Configuration) Quorum() int {
	return (c.VoterCount() / 2) + 1
}

// Contains checks if a member exists in the configuration
func (c *Configuration) Contains(id string) bool {
	_, exists := c.Members[id]
	return exists
}

// IsVoter checks if a member is a voter
func (c *Configuration) IsVoter(id string) bool {
	member, exists := c.Members[id]
	if !exists {
		return false
	}
	return member.Type == Voter
}

// ConfigChangeType represents the type of configuration change
type ConfigChangeType int

const (
	// AddLearnerNode adds a new learner node
	AddLearnerNode ConfigChangeType = iota
	// PromoteNode promotes a learner to voter
	PromoteNode
	// RemoveNode removes a node from the cluster
	RemoveNode
	// DemoteNode demotes a voter to learner
	DemoteNode
)

func (cct ConfigChangeType) String() string {
	switch cct {
	case AddLearnerNode:
		return "AddLearnerNode"
	case PromoteNode:
		return "PromoteNode"
	case RemoveNode:
		return "RemoveNode"
	case DemoteNode:
		return "DemoteNode"
	default:
		return "Unknown"
	}
}

// ConfigChange represents a configuration change operation
type ConfigChange struct {
	Type    ConfigChangeType // Type of configuration change
	NodeID  string           // ID of the node being changed
	Address string           // Address of the node (for add operations)
	Index   uint64           // Log index of this change
	Term    uint64           // Term when this change was proposed
}

// ConfigurationManager manages cluster configuration and tracks changes
type ConfigurationManager struct {
	mu                sync.RWMutex
	current           *Configuration  // Current committed configuration
	pending           *ConfigChange   // Pending configuration change (only one at a time)
	pendingConfig     *Configuration  // What the config will be after pending change
	changeInProgress  bool            // Whether a config change is in progress
	lastChangeIndex   uint64          // Index of last committed config change
}

// NewConfigurationManager creates a new configuration manager
func NewConfigurationManager(initialMembers map[string]string) *ConfigurationManager {
	config := NewConfiguration()
	for id, address := range initialMembers {
		config.AddMember(id, address, Voter)
	}

	return &ConfigurationManager{
		current:          config,
		pending:          nil,
		pendingConfig:    nil,
		changeInProgress: false,
		lastChangeIndex:  0,
	}
}

// GetConfiguration returns a clone of the current configuration
func (cm *ConfigurationManager) GetConfiguration() *Configuration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.current.Clone()
}

// GetPendingChange returns the pending configuration change if any
func (cm *ConfigurationManager) GetPendingChange() *ConfigChange {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if cm.pending == nil {
		return nil
	}
	// Return a copy
	return &ConfigChange{
		Type:    cm.pending.Type,
		NodeID:  cm.pending.NodeID,
		Address: cm.pending.Address,
		Index:   cm.pending.Index,
		Term:    cm.pending.Term,
	}
}

// IsChangeInProgress returns whether a configuration change is in progress
func (cm *ConfigurationManager) IsChangeInProgress() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.changeInProgress
}

// ProposeChange proposes a new configuration change
func (cm *ConfigurationManager) ProposeChange(change *ConfigChange) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.changeInProgress {
		return fmt.Errorf("configuration change already in progress")
	}

	// Validate the change against current configuration
	if err := cm.validateChange(change); err != nil {
		return err
	}

	// Calculate what the new configuration would be
	newConfig := cm.current.Clone()
	if err := cm.applyChangeToConfig(newConfig, change); err != nil {
		return err
	}

	cm.pending = change
	cm.pendingConfig = newConfig
	cm.changeInProgress = true

	return nil
}

// CommitChange commits a pending configuration change
func (cm *ConfigurationManager) CommitChange(index, term uint64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.changeInProgress || cm.pending == nil {
		return fmt.Errorf("no configuration change in progress")
	}

	// Update the configuration
	cm.current = cm.pendingConfig
	cm.current.Index = index
	cm.current.Term = term
	cm.lastChangeIndex = index

	// Clear pending change
	cm.pending = nil
	cm.pendingConfig = nil
	cm.changeInProgress = false

	return nil
}

// AbortChange aborts a pending configuration change
func (cm *ConfigurationManager) AbortChange() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.pending = nil
	cm.pendingConfig = nil
	cm.changeInProgress = false
}

// validateChange validates a proposed configuration change
func (cm *ConfigurationManager) validateChange(change *ConfigChange) error {
	switch change.Type {
	case AddLearnerNode:
		if cm.current.Contains(change.NodeID) {
			return fmt.Errorf("node %s already exists in configuration", change.NodeID)
		}
		if change.Address == "" {
			return fmt.Errorf("address is required for add learner operation")
		}

	case PromoteNode:
		if !cm.current.Contains(change.NodeID) {
			return fmt.Errorf("node %s not found in configuration", change.NodeID)
		}
		if cm.current.IsVoter(change.NodeID) {
			return fmt.Errorf("node %s is already a voter", change.NodeID)
		}

	case RemoveNode:
		if !cm.current.Contains(change.NodeID) {
			return fmt.Errorf("node %s not found in configuration", change.NodeID)
		}
		// Check that we're not removing the last voter
		if cm.current.IsVoter(change.NodeID) && cm.current.VoterCount() <= 1 {
			return fmt.Errorf("cannot remove the last voting member")
		}

	case DemoteNode:
		if !cm.current.Contains(change.NodeID) {
			return fmt.Errorf("node %s not found in configuration", change.NodeID)
		}
		if !cm.current.IsVoter(change.NodeID) {
			return fmt.Errorf("node %s is not a voter", change.NodeID)
		}
		// Check that we're not demoting the last voter
		if cm.current.VoterCount() <= 1 {
			return fmt.Errorf("cannot demote the last voting member")
		}

	default:
		return fmt.Errorf("unknown configuration change type: %v", change.Type)
	}

	return nil
}

// applyChangeToConfig applies a configuration change to a configuration
func (cm *ConfigurationManager) applyChangeToConfig(config *Configuration, change *ConfigChange) error {
	switch change.Type {
	case AddLearnerNode:
		return config.AddMember(change.NodeID, change.Address, Learner)

	case PromoteNode:
		return config.PromoteLearner(change.NodeID)

	case RemoveNode:
		return config.RemoveMember(change.NodeID)

	case DemoteNode:
		member, exists := config.Members[change.NodeID]
		if !exists {
			return fmt.Errorf("member %s not found", change.NodeID)
		}
		member.Type = Learner
		return nil

	default:
		return fmt.Errorf("unknown configuration change type: %v", change.Type)
	}
}

// ApplyConfigChange applies a configuration change (used during log replay)
func (cm *ConfigurationManager) ApplyConfigChange(change *ConfigChange, index, term uint64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	newConfig := cm.current.Clone()
	if err := cm.applyChangeToConfig(newConfig, change); err != nil {
		return err
	}

	newConfig.Index = index
	newConfig.Term = term
	cm.current = newConfig
	cm.lastChangeIndex = index

	// Clear any pending change
	cm.pending = nil
	cm.pendingConfig = nil
	cm.changeInProgress = false

	return nil
}

// SetConfiguration sets the configuration (used during snapshot restoration)
func (cm *ConfigurationManager) SetConfiguration(config *Configuration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.current = config.Clone()
	cm.pending = nil
	cm.pendingConfig = nil
	cm.changeInProgress = false
	cm.lastChangeIndex = config.Index
}

// GetLastChangeIndex returns the index of the last committed config change
func (cm *ConfigurationManager) GetLastChangeIndex() uint64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.lastChangeIndex
}
