package integration

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/cluster"
)

// TestFullMembershipWorkflow tests the complete membership change workflow
func TestFullMembershipWorkflow(t *testing.T) {
	// Initialize a 3-node cluster
	members := map[string]string{
		"node1": "localhost:7001",
		"node2": "localhost:7002",
		"node3": "localhost:7003",
	}
	mm := cluster.NewMembershipManager(members, 100)

	// Verify initial state
	config := mm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Fatalf("expected 3 voters initially, got %d", config.VoterCount())
	}

	// Phase 1: Add a new node as learner
	t.Log("Phase 1: Adding node4 as learner...")
	err := mm.AddLearner("node4", "localhost:7004", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	err = mm.CommitAddLearner("node4", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	// Verify learner was added
	config = mm.GetConfiguration()
	if !config.Contains("node4") {
		t.Error("node4 should be in configuration")
	}

	if config.IsVoter("node4") {
		t.Error("node4 should be a learner, not a voter")
	}

	if len(config.GetLearners()) != 1 {
		t.Errorf("expected 1 learner, got %d", len(config.GetLearners()))
	}

	// Phase 2: Simulate learner catch-up
	t.Log("Phase 2: Simulating learner catch-up...")
	mm.UpdateLearnerProgress("node4", 50, 100, 1024)
	mm.UpdateLearnerProgress("node4", 75, 100, 2048)
	mm.UpdateLearnerProgress("node4", 95, 100, 3072)

	// Verify learner is caught up
	if !mm.IsLearnerCaughtUp("node4") {
		t.Error("node4 should be caught up")
	}

	status := mm.GetLearnerStatus("node4")
	if status == nil {
		t.Fatal("learner status should not be nil")
	}

	if !status.CaughtUp {
		t.Error("learner status should indicate caught up")
	}

	// Phase 3: Promote learner to voter
	t.Log("Phase 3: Promoting node4 to voter...")
	err = mm.PromoteLearner("node4", 2, 1)
	if err != nil {
		t.Fatalf("failed to promote learner: %v", err)
	}

	err = mm.CommitPromoteLearner("node4", 2, 1)
	if err != nil {
		t.Fatalf("failed to commit promote: %v", err)
	}

	// Verify promotion
	config = mm.GetConfiguration()
	if !config.IsVoter("node4") {
		t.Error("node4 should be a voter after promotion")
	}

	if config.VoterCount() != 4 {
		t.Errorf("expected 4 voters after promotion, got %d", config.VoterCount())
	}

	if len(config.GetLearners()) != 0 {
		t.Errorf("expected 0 learners after promotion, got %d", len(config.GetLearners()))
	}

	// Phase 4: Add another learner (for parallel operations test)
	t.Log("Phase 4: Adding node5 as learner...")
	err = mm.AddLearner("node5", "localhost:7005", 3, 1)
	if err != nil {
		t.Fatalf("failed to add second learner: %v", err)
	}

	err = mm.CommitAddLearner("node5", 3, 1)
	if err != nil {
		t.Fatalf("failed to commit second learner: %v", err)
	}

	// Phase 5: Remove an old node
	t.Log("Phase 5: Removing node1...")
	err = mm.RemoveNode("node1", 4, 1)
	if err != nil {
		t.Fatalf("failed to remove node: %v", err)
	}

	err = mm.CommitRemoveNode("node1", 4, 1)
	if err != nil {
		t.Fatalf("failed to commit remove: %v", err)
	}

	// Verify removal
	config = mm.GetConfiguration()
	if config.Contains("node1") {
		t.Error("node1 should be removed from configuration")
	}

	if config.VoterCount() != 3 {
		t.Errorf("expected 3 voters after removal, got %d", config.VoterCount())
	}

	// Phase 6: Promote node5 and verify final state
	t.Log("Phase 6: Promoting node5 to voter...")
	mm.UpdateLearnerProgress("node5", 95, 100, 2048)

	err = mm.PromoteLearner("node5", 5, 1)
	if err != nil {
		t.Fatalf("failed to promote second learner: %v", err)
	}

	err = mm.CommitPromoteLearner("node5", 5, 1)
	if err != nil {
		t.Fatalf("failed to commit second promotion: %v", err)
	}

	// Final verification
	config = mm.GetConfiguration()
	expectedMembers := map[string]bool{
		"node2": true,
		"node3": true,
		"node4": true,
		"node5": true,
	}

	if len(config.Members) != 4 {
		t.Errorf("expected 4 members in final configuration, got %d", len(config.Members))
	}

	for id := range config.Members {
		if !expectedMembers[id] {
			t.Errorf("unexpected member %s in final configuration", id)
		}
		if !config.IsVoter(id) {
			t.Errorf("member %s should be a voter", id)
		}
	}

	if config.VoterCount() != 4 {
		t.Errorf("expected 4 voters in final state, got %d", config.VoterCount())
	}

	if config.Quorum() != 3 {
		t.Errorf("expected quorum of 3, got %d", config.Quorum())
	}

	t.Log("Full membership workflow completed successfully!")
}

// TestReplaceNodeWorkflow tests the node replacement workflow
func TestReplaceNodeWorkflow(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:7001",
		"node2": "localhost:7002",
		"node3": "localhost:7003",
	}
	mm := cluster.NewMembershipManager(members, 10)

	// Initiate replacement
	t.Log("Initiating replacement of node2 with node4...")
	status, err := mm.ReplaceNode("node2", "node4", "localhost:7004")
	if err != nil {
		t.Fatalf("failed to initiate replacement: %v", err)
	}

	if status.OldNodeID != "node2" {
		t.Errorf("expected old node ID node2, got %s", status.OldNodeID)
	}

	if status.NewNodeID != "node4" {
		t.Errorf("expected new node ID node4, got %s", status.NewNodeID)
	}

	// Phase 1: Add new node as learner
	t.Log("Phase 1: Adding node4 as learner...")
	err = mm.AddLearner("node4", "localhost:7004", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	err = mm.CommitAddLearner("node4", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	// Check progress
	progress, err := mm.GetReplaceNodeProgress("node4")
	if err != nil {
		t.Fatalf("failed to get progress: %v", err)
	}

	if progress.Phase != cluster.ReplacePhaseCatchUp {
		t.Errorf("expected phase CatchUp, got %s", progress.Phase)
	}

	// Phase 2: Simulate catch-up
	t.Log("Phase 2: Simulating catch-up...")
	mm.UpdateLearnerProgress("node4", 95, 100, 1024)

	progress, err = mm.GetReplaceNodeProgress("node4")
	if err != nil {
		t.Fatalf("failed to get progress: %v", err)
	}

	if progress.Phase != cluster.ReplacePhasePromote {
		t.Errorf("expected phase Promote, got %s", progress.Phase)
	}

	// Phase 3: Promote new node
	t.Log("Phase 3: Promoting node4...")
	err = mm.PromoteLearner("node4", 2, 1)
	if err != nil {
		t.Fatalf("failed to promote: %v", err)
	}

	err = mm.CommitPromoteLearner("node4", 2, 1)
	if err != nil {
		t.Fatalf("failed to commit promote: %v", err)
	}

	progress, err = mm.GetReplaceNodeProgress("node4")
	if err != nil {
		t.Fatalf("failed to get progress: %v", err)
	}

	if progress.Phase != cluster.ReplacePhaseRemoveOld {
		t.Errorf("expected phase RemoveOld, got %s", progress.Phase)
	}

	// Phase 4: Remove old node
	t.Log("Phase 4: Removing node2...")
	err = mm.RemoveNode("node2", 3, 1)
	if err != nil {
		t.Fatalf("failed to remove old node: %v", err)
	}

	err = mm.CommitRemoveNode("node2", 3, 1)
	if err != nil {
		t.Fatalf("failed to commit remove: %v", err)
	}

	// Verify final state
	config := mm.GetConfiguration()
	if config.Contains("node2") {
		t.Error("node2 should be removed")
	}

	if !config.Contains("node4") {
		t.Error("node4 should be in configuration")
	}

	if !config.IsVoter("node4") {
		t.Error("node4 should be a voter")
	}

	if config.VoterCount() != 3 {
		t.Errorf("expected 3 voters, got %d", config.VoterCount())
	}

	t.Log("Node replacement workflow completed successfully!")
}

// TestConcurrentMembershipChanges verifies that concurrent changes are prevented
func TestConcurrentMembershipChanges(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:7001",
		"node2": "localhost:7002",
	}
	mm := cluster.NewMembershipManager(members, 100)

	// Start first change
	err := mm.AddLearner("node3", "localhost:7003", 1, 1)
	if err != nil {
		t.Fatalf("failed to add first learner: %v", err)
	}

	// Try to start second change while first is in progress
	err = mm.AddLearner("node4", "localhost:7004", 1, 1)
	if err == nil {
		t.Error("expected error when starting concurrent change")
	}

	// Verify only first change is in progress
	if !mm.IsChangeInProgress() {
		t.Error("change should be in progress")
	}

	pending := mm.GetPendingChange()
	if pending == nil {
		t.Fatal("pending change should not be nil")
	}

	if pending.NodeID != "node3" {
		t.Errorf("expected pending change for node3, got %s", pending.NodeID)
	}

	// Complete first change
	err = mm.CommitAddLearner("node3", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit first change: %v", err)
	}

	// Now second change should be allowed
	err = mm.AddLearner("node4", "localhost:7004", 2, 1)
	if err != nil {
		t.Errorf("second change should be allowed after first completes: %v", err)
	}

	t.Log("Concurrent change prevention working correctly!")
}

// TestLearnerPromotionCallback tests the automatic promotion callback
func TestLearnerPromotionCallback(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:7001",
	}
	mm := cluster.NewMembershipManager(members, 10)

	callbackCalled := false
	var callbackNodeID string

	// Set promotion callback
	mm.SetPromotionCallback(func(nodeID string) error {
		callbackCalled = true
		callbackNodeID = nodeID
		return nil
	})

	// Add learner
	err := mm.AddLearner("node2", "localhost:7002", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	err = mm.CommitAddLearner("node2", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	// Learner not caught up yet - callback should not be called
	mm.UpdateLearnerProgress("node2", 50, 100, 1024)
	time.Sleep(100 * time.Millisecond) // Give goroutine time to run if it would

	if callbackCalled {
		t.Error("callback should not be called when learner not caught up")
	}

	// Update to caught up - callback should be called
	mm.UpdateLearnerProgress("node2", 95, 100, 2048)
	time.Sleep(100 * time.Millisecond) // Give goroutine time to run

	if !callbackCalled {
		t.Error("callback should be called when learner catches up")
	}

	if callbackNodeID != "node2" {
		t.Errorf("expected callback for node2, got %s", callbackNodeID)
	}

	t.Log("Promotion callback working correctly!")
}

// TestMembershipSerialization tests saving and restoring membership state
func TestMembershipSerialization(t *testing.T) {
	// Create initial membership
	members := map[string]string{
		"node1": "localhost:7001",
		"node2": "localhost:7002",
		"node3": "localhost:7003",
	}
	mm1 := cluster.NewMembershipManager(members, 100)

	// Add a learner
	err := mm1.AddLearner("node4", "localhost:7004", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	err = mm1.CommitAddLearner("node4", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	// Serialize
	data, err := mm1.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Create new membership manager and deserialize
	mm2 := cluster.NewMembershipManager(map[string]string{}, 100)
	err = mm2.Deserialize(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify state was restored
	config1 := mm1.GetConfiguration()
	config2 := mm2.GetConfiguration()

	if len(config1.Members) != len(config2.Members) {
		t.Errorf("member count mismatch: %d vs %d", len(config1.Members), len(config2.Members))
	}

	for id, member1 := range config1.Members {
		member2, exists := config2.Members[id]
		if !exists {
			t.Errorf("member %s missing after deserialization", id)
			continue
		}

		if member1.Address != member2.Address {
			t.Errorf("address mismatch for %s: %s vs %s", id, member1.Address, member2.Address)
		}

		if member1.Type != member2.Type {
			t.Errorf("type mismatch for %s: %v vs %v", id, member1.Type, member2.Type)
		}
	}

	t.Log("Serialization working correctly!")
}
