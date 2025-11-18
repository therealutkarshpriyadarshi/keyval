package cluster

import (
	"testing"
	"time"
)

func TestMembershipManager_AddLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	mm := NewMembershipManager(members, 100)

	err := mm.AddLearner("node4", "localhost:5004", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	if !mm.IsChangeInProgress() {
		t.Error("change should be in progress")
	}

	status := mm.GetLearnerStatus("node4")
	if status == nil {
		t.Fatal("learner status not created")
	}

	if status.CaughtUp {
		t.Error("learner should not be caught up initially")
	}
}

func TestMembershipManager_CommitAddLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	err := mm.AddLearner("node2", "localhost:5002", 1, 1)
	if err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	err = mm.CommitAddLearner("node2", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	if mm.IsChangeInProgress() {
		t.Error("change should not be in progress after commit")
	}

	config := mm.GetConfiguration()
	if !config.Contains("node2") {
		t.Error("learner not in configuration after commit")
	}

	if config.IsVoter("node2") {
		t.Error("node2 should be a learner, not a voter")
	}
}

func TestMembershipManager_UpdateLearnerProgress(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 10) // Small gap for testing

	mm.AddLearner("node2", "localhost:5002", 1, 1)
	mm.CommitAddLearner("node2", 1, 1)

	// Initially not caught up
	if mm.IsLearnerCaughtUp("node2") {
		t.Error("learner should not be caught up initially")
	}

	// Update progress - still not caught up (gap > 10)
	mm.UpdateLearnerProgress("node2", 50, 100, 1024)

	status := mm.GetLearnerStatus("node2")
	if status.MatchIndex != 50 {
		t.Errorf("expected matchIndex 50, got %d", status.MatchIndex)
	}

	if status.BytesReplicated != 1024 {
		t.Errorf("expected bytesReplicated 1024, got %d", status.BytesReplicated)
	}

	if mm.IsLearnerCaughtUp("node2") {
		t.Error("learner should not be caught up yet (gap = 50)")
	}

	// Update progress - now caught up (gap <= 10)
	mm.UpdateLearnerProgress("node2", 95, 100, 512)

	if !mm.IsLearnerCaughtUp("node2") {
		t.Error("learner should be caught up now (gap = 5)")
	}

	status = mm.GetLearnerStatus("node2")
	if status.BytesReplicated != 1536 {
		t.Errorf("expected bytesReplicated 1536, got %d", status.BytesReplicated)
	}
}

func TestMembershipManager_PromoteLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 10)

	// Add and commit learner
	mm.AddLearner("node2", "localhost:5002", 1, 1)
	mm.CommitAddLearner("node2", 1, 1)

	// Try to promote before caught up - should fail
	err := mm.PromoteLearner("node2", 2, 1)
	if err == nil {
		t.Error("expected error when promoting learner that's not caught up")
	}

	// Make learner caught up
	mm.UpdateLearnerProgress("node2", 95, 100, 0)

	// Now promote should succeed
	err = mm.PromoteLearner("node2", 2, 1)
	if err != nil {
		t.Fatalf("failed to promote learner: %v", err)
	}

	err = mm.CommitPromoteLearner("node2", 2, 1)
	if err != nil {
		t.Fatalf("failed to commit promote learner: %v", err)
	}

	config := mm.GetConfiguration()
	if !config.IsVoter("node2") {
		t.Error("node2 should be a voter after promotion")
	}

	// Learner status should be removed
	status := mm.GetLearnerStatus("node2")
	if status != nil {
		t.Error("learner status should be removed after promotion")
	}
}

func TestMembershipManager_RemoveNode(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	mm := NewMembershipManager(members, 100)

	err := mm.RemoveNode("node3", 1, 1)
	if err != nil {
		t.Fatalf("failed to remove node: %v", err)
	}

	err = mm.CommitRemoveNode("node3", 1, 1)
	if err != nil {
		t.Fatalf("failed to commit remove node: %v", err)
	}

	config := mm.GetConfiguration()
	if config.Contains("node3") {
		t.Error("node3 should be removed from configuration")
	}

	if config.VoterCount() != 2 {
		t.Errorf("expected 2 voters, got %d", config.VoterCount())
	}
}

func TestMembershipManager_RemoveLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	// Add learner
	mm.AddLearner("learner1", "localhost:5002", 1, 1)
	mm.CommitAddLearner("learner1", 1, 1)

	// Remove learner
	err := mm.RemoveNode("learner1", 2, 1)
	if err != nil {
		t.Fatalf("failed to remove learner: %v", err)
	}

	err = mm.CommitRemoveNode("learner1", 2, 1)
	if err != nil {
		t.Fatalf("failed to commit remove learner: %v", err)
	}

	config := mm.GetConfiguration()
	if config.Contains("learner1") {
		t.Error("learner1 should be removed from configuration")
	}

	// Learner status should be removed
	status := mm.GetLearnerStatus("learner1")
	if status != nil {
		t.Error("learner status should be removed")
	}
}

func TestMembershipManager_AbortChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	mm.AddLearner("node2", "localhost:5002", 1, 1)

	mm.AbortChange()

	if mm.IsChangeInProgress() {
		t.Error("change should not be in progress after abort")
	}

	// Learner status should be removed on abort
	status := mm.GetLearnerStatus("node2")
	if status != nil {
		t.Error("learner status should be removed on abort")
	}
}

func TestMembershipManager_PromotionCallback(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 10)

	callbackCalled := make(chan string, 1)
	mm.SetPromotionCallback(func(nodeID string) error {
		callbackCalled <- nodeID
		return nil
	})

	mm.AddLearner("node2", "localhost:5002", 1, 1)
	mm.CommitAddLearner("node2", 1, 1)

	// Update progress to make learner caught up
	mm.UpdateLearnerProgress("node2", 95, 100, 0)

	// Callback should be called
	select {
	case nodeID := <-callbackCalled:
		if nodeID != "node2" {
			t.Errorf("expected callback with node2, got %s", nodeID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("promotion callback was not called")
	}
}

func TestMembershipManager_ApplyConfigChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	// Apply add learner
	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node2",
		Address: "localhost:5002",
	}
	err := mm.ApplyConfigChange(change, 1, 1)
	if err != nil {
		t.Fatalf("failed to apply add learner: %v", err)
	}

	config := mm.GetConfiguration()
	if !config.Contains("node2") {
		t.Error("node2 not in configuration after apply")
	}

	status := mm.GetLearnerStatus("node2")
	if status == nil {
		t.Error("learner status not created")
	}

	// Apply promote
	change = &ConfigChange{
		Type:   PromoteNode,
		NodeID: "node2",
	}
	err = mm.ApplyConfigChange(change, 2, 1)
	if err != nil {
		t.Fatalf("failed to apply promote: %v", err)
	}

	config = mm.GetConfiguration()
	if !config.IsVoter("node2") {
		t.Error("node2 should be voter after promote")
	}

	status = mm.GetLearnerStatus("node2")
	if status != nil {
		t.Error("learner status should be removed after promote")
	}
}

func TestMembershipManager_SetConfiguration(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	// Create a new configuration
	config := NewConfiguration()
	config.AddMember("node2", "localhost:5002", Voter)
	config.AddMember("node3", "localhost:5003", Learner)
	config.Index = 100
	config.Term = 5

	mm.SetConfiguration(config)

	newConfig := mm.GetConfiguration()
	if newConfig.VoterCount() != 1 {
		t.Errorf("expected 1 voter, got %d", newConfig.VoterCount())
	}

	if len(newConfig.GetLearners()) != 1 {
		t.Errorf("expected 1 learner, got %d", len(newConfig.GetLearners()))
	}

	// Learner status should be created
	status := mm.GetLearnerStatus("node3")
	if status == nil {
		t.Error("learner status not created for node3")
	}
}

func TestMembershipManager_GetAllLearnerStatuses(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	mm := NewMembershipManager(members, 100)

	mm.AddLearner("learner1", "localhost:5002", 1, 1)
	mm.CommitAddLearner("learner1", 1, 1)

	mm.AddLearner("learner2", "localhost:5003", 2, 1)
	mm.CommitAddLearner("learner2", 2, 1)

	statuses := mm.GetAllLearnerStatuses()
	if len(statuses) != 2 {
		t.Errorf("expected 2 learner statuses, got %d", len(statuses))
	}

	if _, exists := statuses["learner1"]; !exists {
		t.Error("learner1 status not found")
	}

	if _, exists := statuses["learner2"]; !exists {
		t.Error("learner2 status not found")
	}
}

func TestMembershipManager_Serialize(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
	}
	mm := NewMembershipManager(members, 100)

	data, err := mm.Serialize()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	if len(data) == 0 {
		t.Error("serialized data is empty")
	}

	// Deserialize into new manager
	mm2 := NewMembershipManager(nil, 100)
	err = mm2.Deserialize(data)
	if err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	config := mm2.GetConfiguration()
	if config.VoterCount() != 2 {
		t.Errorf("expected 2 voters after deserialize, got %d", config.VoterCount())
	}

	if !config.Contains("node1") || !config.Contains("node2") {
		t.Error("members not restored after deserialize")
	}
}

func TestMembershipManager_FullWorkflow(t *testing.T) {
	// Start with 3 voters
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	mm := NewMembershipManager(members, 10)

	config := mm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Fatalf("expected 3 voters initially, got %d", config.VoterCount())
	}

	// Step 1: Add a learner
	if err := mm.AddLearner("node4", "localhost:5004", 1, 1); err != nil {
		t.Fatalf("failed to add learner: %v", err)
	}

	if err := mm.CommitAddLearner("node4", 1, 1); err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	config = mm.GetConfiguration()
	if len(config.GetLearners()) != 1 {
		t.Errorf("expected 1 learner, got %d", len(config.GetLearners()))
	}

	// Step 2: Update learner progress to make it caught up
	mm.UpdateLearnerProgress("node4", 95, 100, 1024)

	if !mm.IsLearnerCaughtUp("node4") {
		t.Error("node4 should be caught up")
	}

	// Step 3: Promote the learner
	if err := mm.PromoteLearner("node4", 2, 1); err != nil {
		t.Fatalf("failed to promote learner: %v", err)
	}

	if err := mm.CommitPromoteLearner("node4", 2, 1); err != nil {
		t.Fatalf("failed to commit promote: %v", err)
	}

	config = mm.GetConfiguration()
	if config.VoterCount() != 4 {
		t.Errorf("expected 4 voters after promotion, got %d", config.VoterCount())
	}

	if config.IsVoter("node4") == false {
		t.Error("node4 should be a voter")
	}

	// Step 4: Remove a node
	if err := mm.RemoveNode("node1", 3, 1); err != nil {
		t.Fatalf("failed to remove node: %v", err)
	}

	if err := mm.CommitRemoveNode("node1", 3, 1); err != nil {
		t.Fatalf("failed to commit remove: %v", err)
	}

	config = mm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Errorf("expected 3 voters after removal, got %d", config.VoterCount())
	}

	if config.Contains("node1") {
		t.Error("node1 should be removed")
	}
}
