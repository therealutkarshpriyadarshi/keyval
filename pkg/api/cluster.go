package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/cluster"
)

// ClusterAPI provides cluster management and query operations
type ClusterAPI struct {
	mu                sync.RWMutex
	nodeID            string
	membershipManager *cluster.MembershipManager
	getCurrentTerm    func() uint64
	getLeaderID       func() string
	getState          func() NodeState
	getLastApplied    func() uint64
	getCommitIndex    func() uint64
}

// NodeState represents the current state of a Raft node
type NodeState string

const (
	StateFollower  NodeState = "Follower"
	StateCandidate NodeState = "Candidate"
	StateLeader    NodeState = "Leader"
)

// NewClusterAPI creates a new cluster API
func NewClusterAPI(nodeID string, mm *cluster.MembershipManager) *ClusterAPI {
	return &ClusterAPI{
		nodeID:            nodeID,
		membershipManager: mm,
	}
}

// SetCallbacks sets the callback functions for querying node state
func (ca *ClusterAPI) SetCallbacks(
	getCurrentTerm func() uint64,
	getLeaderID func() string,
	getState func() NodeState,
	getLastApplied func() uint64,
	getCommitIndex func() uint64,
) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.getCurrentTerm = getCurrentTerm
	ca.getLeaderID = getLeaderID
	ca.getState = getState
	ca.getLastApplied = getLastApplied
	ca.getCommitIndex = getCommitIndex
}

// ClusterInfo represents overall cluster information
type ClusterInfo struct {
	LeaderID     string          `json:"leader_id"`
	CurrentTerm  uint64          `json:"current_term"`
	Members      []*MemberInfo   `json:"members"`
	MemberCount  int             `json:"member_count"`
	VoterCount   int             `json:"voter_count"`
	LearnerCount int             `json:"learner_count"`
	Quorum       int             `json:"quorum"`
	Healthy      bool            `json:"healthy"`
	Timestamp    time.Time       `json:"timestamp"`
}

// MemberInfo represents information about a cluster member
type MemberInfo struct {
	ID            string               `json:"id"`
	Address       string               `json:"address"`
	Type          string               `json:"type"` // "Voter" or "Learner"
	State         NodeState            `json:"state,omitempty"`
	IsLeader      bool                 `json:"is_leader"`
	IsSelf        bool                 `json:"is_self"`
	AddedAt       time.Time            `json:"added_at"`
	LastContact   *time.Time           `json:"last_contact,omitempty"`
	LearnerStatus *cluster.LearnerStatus `json:"learner_status,omitempty"`
}

// ClusterHealth represents cluster health status
type ClusterHealth struct {
	Healthy         bool               `json:"healthy"`
	LeaderPresent   bool               `json:"leader_present"`
	QuorumAvailable bool               `json:"quorum_available"`
	MembersHealthy  int                `json:"members_healthy"`
	MembersTotal    int                `json:"members_total"`
	Issues          []string           `json:"issues,omitempty"`
	Timestamp       time.Time          `json:"timestamp"`
}

// GetClusterInfo returns comprehensive cluster information
func (ca *ClusterAPI) GetClusterInfo() (*ClusterInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	config := ca.membershipManager.GetConfiguration()
	if config == nil {
		return nil, fmt.Errorf("no configuration available")
	}

	var leaderID string
	var currentTerm uint64
	if ca.getLeaderID != nil {
		leaderID = ca.getLeaderID()
	}
	if ca.getCurrentTerm != nil {
		currentTerm = ca.getCurrentTerm()
	}

	members := make([]*MemberInfo, 0, len(config.Members))
	for id, member := range config.Members {
		info := &MemberInfo{
			ID:       id,
			Address:  member.Address,
			Type:     member.Type.String(),
			IsLeader: id == leaderID,
			IsSelf:   id == ca.nodeID,
			AddedAt:  member.AddedAt,
		}

		// Add node state if this is the current node
		if id == ca.nodeID && ca.getState != nil {
			info.State = ca.getState()
		}

		// Add learner status if applicable
		if member.Type == cluster.Learner {
			if status := ca.membershipManager.GetLearnerStatus(id); status != nil {
				info.LearnerStatus = status
				info.LastContact = &status.LastContact
			}
		}

		members = append(members, info)
	}

	return &ClusterInfo{
		LeaderID:     leaderID,
		CurrentTerm:  currentTerm,
		Members:      members,
		MemberCount:  len(config.Members),
		VoterCount:   config.VoterCount(),
		LearnerCount: len(config.GetLearners()),
		Quorum:       config.Quorum(),
		Healthy:      leaderID != "",
		Timestamp:    time.Now(),
	}, nil
}

// ListMembers returns a list of all cluster members
func (ca *ClusterAPI) ListMembers() ([]*MemberInfo, error) {
	info, err := ca.GetClusterInfo()
	if err != nil {
		return nil, err
	}
	return info.Members, nil
}

// GetMember returns information about a specific member
func (ca *ClusterAPI) GetMember(nodeID string) (*MemberInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	config := ca.membershipManager.GetConfiguration()
	member, exists := config.Members[nodeID]
	if !exists {
		return nil, fmt.Errorf("member %s not found", nodeID)
	}

	var leaderID string
	if ca.getLeaderID != nil {
		leaderID = ca.getLeaderID()
	}

	info := &MemberInfo{
		ID:       nodeID,
		Address:  member.Address,
		Type:     member.Type.String(),
		IsLeader: nodeID == leaderID,
		IsSelf:   nodeID == ca.nodeID,
		AddedAt:  member.AddedAt,
	}

	if nodeID == ca.nodeID && ca.getState != nil {
		info.State = ca.getState()
	}

	if member.Type == cluster.Learner {
		if status := ca.membershipManager.GetLearnerStatus(nodeID); status != nil {
			info.LearnerStatus = status
			info.LastContact = &status.LastContact
		}
	}

	return info, nil
}

// GetLeader returns information about the current leader
func (ca *ClusterAPI) GetLeader() (*MemberInfo, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var leaderID string
	if ca.getLeaderID != nil {
		leaderID = ca.getLeaderID()
	}

	if leaderID == "" {
		return nil, fmt.Errorf("no leader elected")
	}

	return ca.GetMember(leaderID)
}

// GetClusterHealth returns cluster health status
func (ca *ClusterAPI) GetClusterHealth() (*ClusterHealth, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	config := ca.membershipManager.GetConfiguration()
	if config == nil {
		return nil, fmt.Errorf("no configuration available")
	}

	var leaderID string
	if ca.getLeaderID != nil {
		leaderID = ca.getLeaderID()
	}

	health := &ClusterHealth{
		LeaderPresent:   leaderID != "",
		QuorumAvailable: true, // Assume quorum unless we can check connectivity
		MembersHealthy:  len(config.Members),
		MembersTotal:    len(config.Members),
		Issues:          make([]string, 0),
		Timestamp:       time.Now(),
	}

	// Check for leader
	if !health.LeaderPresent {
		health.Healthy = false
		health.Issues = append(health.Issues, "No leader elected")
	}

	// Check voter count
	if config.VoterCount() == 0 {
		health.Healthy = false
		health.QuorumAvailable = false
		health.Issues = append(health.Issues, "No voting members in cluster")
	}

	// Check if we have enough voters for fault tolerance
	if config.VoterCount() < 3 {
		health.Issues = append(health.Issues, fmt.Sprintf("Only %d voter(s), cluster cannot tolerate failures", config.VoterCount()))
	}

	// If no issues found, cluster is healthy
	if len(health.Issues) == 0 {
		health.Healthy = true
	}

	return health, nil
}

// GetNodeStatus returns the status of the current node
func (ca *ClusterAPI) GetNodeStatus() (*NodeStatus, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var state NodeState
	var leaderID string
	var currentTerm uint64
	var lastApplied uint64
	var commitIndex uint64

	if ca.getState != nil {
		state = ca.getState()
	}
	if ca.getLeaderID != nil {
		leaderID = ca.getLeaderID()
	}
	if ca.getCurrentTerm != nil {
		currentTerm = ca.getCurrentTerm()
	}
	if ca.getLastApplied != nil {
		lastApplied = ca.getLastApplied()
	}
	if ca.getCommitIndex != nil {
		commitIndex = ca.getCommitIndex()
	}

	config := ca.membershipManager.GetConfiguration()
	member, exists := config.Members[ca.nodeID]
	if !exists {
		return nil, fmt.Errorf("current node not in configuration")
	}

	status := &NodeStatus{
		NodeID:       ca.nodeID,
		Address:      member.Address,
		State:        state,
		IsLeader:     ca.nodeID == leaderID,
		LeaderID:     leaderID,
		CurrentTerm:  currentTerm,
		CommitIndex:  commitIndex,
		LastApplied:  lastApplied,
		MemberType:   member.Type.String(),
		Timestamp:    time.Now(),
	}

	return status, nil
}

// NodeStatus represents the status of a single node
type NodeStatus struct {
	NodeID      string    `json:"node_id"`
	Address     string    `json:"address"`
	State       NodeState `json:"state"`
	IsLeader    bool      `json:"is_leader"`
	LeaderID    string    `json:"leader_id"`
	CurrentTerm uint64    `json:"current_term"`
	CommitIndex uint64    `json:"commit_index"`
	LastApplied uint64    `json:"last_applied"`
	MemberType  string    `json:"member_type"`
	Timestamp   time.Time `json:"timestamp"`
}

// IsConfigChangeInProgress returns whether a configuration change is in progress
func (ca *ClusterAPI) IsConfigChangeInProgress() bool {
	return ca.membershipManager.IsChangeInProgress()
}

// GetPendingConfigChange returns information about any pending configuration change
func (ca *ClusterAPI) GetPendingConfigChange() *cluster.ConfigChange {
	return ca.membershipManager.GetPendingChange()
}
