package api

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/cluster"
)

func TestNewClusterAPI(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	if ca.nodeID != "node1" {
		t.Errorf("expected nodeID node1, got %s", ca.nodeID)
	}

	if ca.membershipManager == nil {
		t.Error("membership manager should not be nil")
	}
}

func TestClusterAPI_GetClusterInfo(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// Set callbacks
	ca.SetCallbacks(
		func() uint64 { return 5 },           // getCurrentTerm
		func() string { return "node1" },     // getLeaderID
		func() NodeState { return StateLeader }, // getState
		func() uint64 { return 100 },         // getLastApplied
		func() uint64 { return 100 },         // getCommitIndex
	)

	info, err := ca.GetClusterInfo()
	if err != nil {
		t.Fatalf("failed to get cluster info: %v", err)
	}

	if info.LeaderID != "node1" {
		t.Errorf("expected leader node1, got %s", info.LeaderID)
	}

	if info.CurrentTerm != 5 {
		t.Errorf("expected term 5, got %d", info.CurrentTerm)
	}

	if info.MemberCount != 3 {
		t.Errorf("expected 3 members, got %d", info.MemberCount)
	}

	if info.VoterCount != 3 {
		t.Errorf("expected 3 voters, got %d", info.VoterCount)
	}

	if info.LearnerCount != 0 {
		t.Errorf("expected 0 learners, got %d", info.LearnerCount)
	}

	if info.Quorum != 2 {
		t.Errorf("expected quorum 2, got %d", info.Quorum)
	}

	if !info.Healthy {
		t.Error("cluster should be healthy")
	}

	// Check member info
	foundSelf := false
	foundLeader := false
	for _, member := range info.Members {
		if member.ID == "node1" {
			foundSelf = true
			if !member.IsSelf {
				t.Error("node1 should be marked as self")
			}
			if !member.IsLeader {
				t.Error("node1 should be marked as leader")
			}
			if member.State != StateLeader {
				t.Errorf("expected state Leader, got %s", member.State)
			}
			foundLeader = true
		}
	}

	if !foundSelf {
		t.Error("did not find self in member list")
	}

	if !foundLeader {
		t.Error("did not find leader in member list")
	}
}

func TestClusterAPI_GetClusterInfo_WithLearners(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 10)
	ca := NewClusterAPI("node1", mm)

	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "node1" },
		func() NodeState { return StateLeader },
		func() uint64 { return 50 },
		func() uint64 { return 50 },
	)

	// Add a learner
	mm.AddLearner("node2", "localhost:5002", 1, 1)
	mm.CommitAddLearner("node2", 1, 1)
	mm.UpdateLearnerProgress("node2", 40, 50, 1024)

	info, err := ca.GetClusterInfo()
	if err != nil {
		t.Fatalf("failed to get cluster info: %v", err)
	}

	if info.VoterCount != 1 {
		t.Errorf("expected 1 voter, got %d", info.VoterCount)
	}

	if info.LearnerCount != 1 {
		t.Errorf("expected 1 learner, got %d", info.LearnerCount)
	}

	// Find the learner in members
	var learnerFound bool
	for _, member := range info.Members {
		if member.ID == "node2" {
			learnerFound = true
			if member.Type != "Learner" {
				t.Errorf("expected type Learner, got %s", member.Type)
			}
			if member.LearnerStatus == nil {
				t.Error("learner status should not be nil")
			} else {
				if member.LearnerStatus.MatchIndex != 40 {
					t.Errorf("expected matchIndex 40, got %d", member.LearnerStatus.MatchIndex)
				}
			}
		}
	}

	if !learnerFound {
		t.Error("learner not found in members")
	}
}

func TestClusterAPI_ListMembers(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	memberList, err := ca.ListMembers()
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if len(memberList) != 2 {
		t.Errorf("expected 2 members, got %d", len(memberList))
	}
}

func TestClusterAPI_GetMember(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "node1" },
		func() NodeState { return StateLeader },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	// Get existing member
	member, err := ca.GetMember("node2")
	if err != nil {
		t.Fatalf("failed to get member: %v", err)
	}

	if member.ID != "node2" {
		t.Errorf("expected ID node2, got %s", member.ID)
	}

	if member.Address != "localhost:5002" {
		t.Errorf("expected address localhost:5002, got %s", member.Address)
	}

	// Get self
	self, err := ca.GetMember("node1")
	if err != nil {
		t.Fatalf("failed to get self: %v", err)
	}

	if !self.IsSelf {
		t.Error("node1 should be marked as self")
	}

	if self.State != StateLeader {
		t.Errorf("expected state Leader, got %s", self.State)
	}

	// Get non-existent member
	_, err = ca.GetMember("node99")
	if err == nil {
		t.Error("expected error for non-existent member")
	}
}

func TestClusterAPI_GetLeader(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// Set leader callback
	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "node2" },
		func() NodeState { return StateFollower },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	leader, err := ca.GetLeader()
	if err != nil {
		t.Fatalf("failed to get leader: %v", err)
	}

	if leader.ID != "node2" {
		t.Errorf("expected leader node2, got %s", leader.ID)
	}

	if !leader.IsLeader {
		t.Error("node2 should be marked as leader")
	}

	// Test no leader
	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "" },
		func() NodeState { return StateFollower },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	_, err = ca.GetLeader()
	if err == nil {
		t.Error("expected error when no leader elected")
	}
}

func TestClusterAPI_GetClusterHealth(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// Test healthy cluster
	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "node1" },
		func() NodeState { return StateLeader },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	health, err := ca.GetClusterHealth()
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}

	if !health.Healthy {
		t.Error("cluster should be healthy")
	}

	if !health.LeaderPresent {
		t.Error("leader should be present")
	}

	if health.MembersTotal != 3 {
		t.Errorf("expected 3 total members, got %d", health.MembersTotal)
	}

	// Test cluster with no leader
	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "" },
		func() NodeState { return StateFollower },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	health, err = ca.GetClusterHealth()
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}

	if health.Healthy {
		t.Error("cluster should not be healthy without leader")
	}

	if health.LeaderPresent {
		t.Error("leader should not be present")
	}

	if len(health.Issues) == 0 {
		t.Error("expected health issues")
	}
}

func TestClusterAPI_GetClusterHealth_SingleNode(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	ca.SetCallbacks(
		func() uint64 { return 1 },
		func() string { return "node1" },
		func() NodeState { return StateLeader },
		func() uint64 { return 10 },
		func() uint64 { return 10 },
	)

	health, err := ca.GetClusterHealth()
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}

	// Single node cluster has a warning but can still be healthy
	if !health.LeaderPresent {
		t.Error("leader should be present")
	}

	// Should have a warning about fault tolerance
	foundWarning := false
	for _, issue := range health.Issues {
		if issue == "Only 1 voter(s), cluster cannot tolerate failures" {
			foundWarning = true
			break
		}
	}

	if !foundWarning {
		t.Error("expected warning about single voter")
	}
}

func TestClusterAPI_GetNodeStatus(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	ca.SetCallbacks(
		func() uint64 { return 5 },
		func() string { return "node1" },
		func() NodeState { return StateLeader },
		func() uint64 { return 100 },
		func() uint64 { return 105 },
	)

	status, err := ca.GetNodeStatus()
	if err != nil {
		t.Fatalf("failed to get node status: %v", err)
	}

	if status.NodeID != "node1" {
		t.Errorf("expected nodeID node1, got %s", status.NodeID)
	}

	if status.State != StateLeader {
		t.Errorf("expected state Leader, got %s", status.State)
	}

	if !status.IsLeader {
		t.Error("node should be leader")
	}

	if status.CurrentTerm != 5 {
		t.Errorf("expected term 5, got %d", status.CurrentTerm)
	}

	if status.LastApplied != 100 {
		t.Errorf("expected lastApplied 100, got %d", status.LastApplied)
	}

	if status.CommitIndex != 105 {
		t.Errorf("expected commitIndex 105, got %d", status.CommitIndex)
	}

	if status.MemberType != "Voter" {
		t.Errorf("expected type Voter, got %s", status.MemberType)
	}
}

func TestClusterAPI_IsConfigChangeInProgress(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// No change initially
	if ca.IsConfigChangeInProgress() {
		t.Error("no change should be in progress initially")
	}

	// Start a change
	mm.AddLearner("node2", "localhost:5002", 1, 1)

	if !ca.IsConfigChangeInProgress() {
		t.Error("change should be in progress")
	}

	// Commit the change
	mm.CommitAddLearner("node2", 1, 1)

	if ca.IsConfigChangeInProgress() {
		t.Error("no change should be in progress after commit")
	}
}

func TestClusterAPI_GetPendingConfigChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// No pending change initially
	if ca.GetPendingConfigChange() != nil {
		t.Error("no pending change should exist initially")
	}

	// Start a change
	mm.AddLearner("node2", "localhost:5002", 1, 1)

	change := ca.GetPendingConfigChange()
	if change == nil {
		t.Fatal("pending change should exist")
	}

	if change.Type != cluster.AddLearnerNode {
		t.Errorf("expected AddLearnerNode, got %v", change.Type)
	}

	if change.NodeID != "node2" {
		t.Errorf("expected nodeID node2, got %s", change.NodeID)
	}
}

func TestMemberInfo_WithLearnerStatus(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := cluster.NewMembershipManager(members, 100)
	ca := NewClusterAPI("node1", mm)

	// Add learner
	mm.AddLearner("node2", "localhost:5002", 1, 1)
	mm.CommitAddLearner("node2", 1, 1)
	mm.UpdateLearnerProgress("node2", 50, 100, 2048)

	member, err := ca.GetMember("node2")
	if err != nil {
		t.Fatalf("failed to get member: %v", err)
	}

	if member.LearnerStatus == nil {
		t.Fatal("learner status should not be nil")
	}

	if member.LearnerStatus.MatchIndex != 50 {
		t.Errorf("expected matchIndex 50, got %d", member.LearnerStatus.MatchIndex)
	}

	if member.LearnerStatus.BytesReplicated != 2048 {
		t.Errorf("expected bytesReplicated 2048, got %d", member.LearnerStatus.BytesReplicated)
	}

	if member.LastContact == nil {
		t.Error("last contact should not be nil for learner")
	}

	// Verify last contact is recent
	if time.Since(*member.LastContact) > time.Second {
		t.Error("last contact should be recent")
	}
}
