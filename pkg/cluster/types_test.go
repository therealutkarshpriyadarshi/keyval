package cluster

import (
	"testing"
)

func TestConfiguration_AddMember(t *testing.T) {
	config := NewConfiguration()

	// Add a voter
	err := config.AddMember("node1", "localhost:5001", Voter)
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	if !config.Contains("node1") {
		t.Error("member not found after adding")
	}

	if !config.IsVoter("node1") {
		t.Error("member should be a voter")
	}

	// Try to add duplicate
	err = config.AddMember("node1", "localhost:5001", Voter)
	if err == nil {
		t.Error("expected error when adding duplicate member")
	}
}

func TestConfiguration_RemoveMember(t *testing.T) {
	config := NewConfiguration()
	config.AddMember("node1", "localhost:5001", Voter)
	config.AddMember("node2", "localhost:5002", Voter)

	err := config.RemoveMember("node1")
	if err != nil {
		t.Fatalf("failed to remove member: %v", err)
	}

	if config.Contains("node1") {
		t.Error("member should be removed")
	}

	// Try to remove non-existent member
	err = config.RemoveMember("node3")
	if err == nil {
		t.Error("expected error when removing non-existent member")
	}
}

func TestConfiguration_PromoteLearner(t *testing.T) {
	config := NewConfiguration()
	config.AddMember("learner1", "localhost:5001", Learner)

	err := config.PromoteLearner("learner1")
	if err != nil {
		t.Fatalf("failed to promote learner: %v", err)
	}

	if !config.IsVoter("learner1") {
		t.Error("learner should be promoted to voter")
	}

	// Try to promote a voter
	err = config.PromoteLearner("learner1")
	if err == nil {
		t.Error("expected error when promoting a voter")
	}
}

func TestConfiguration_VoterCount(t *testing.T) {
	config := NewConfiguration()
	config.AddMember("node1", "localhost:5001", Voter)
	config.AddMember("node2", "localhost:5002", Voter)
	config.AddMember("learner1", "localhost:5003", Learner)

	if config.VoterCount() != 2 {
		t.Errorf("expected 2 voters, got %d", config.VoterCount())
	}
}

func TestConfiguration_Quorum(t *testing.T) {
	tests := []struct {
		name       string
		voterCount int
		wantQuorum int
	}{
		{"1 voter", 1, 1},
		{"2 voters", 2, 2},
		{"3 voters", 3, 2},
		{"4 voters", 4, 3},
		{"5 voters", 5, 3},
		{"6 voters", 6, 4},
		{"7 voters", 7, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfiguration()
			for i := 0; i < tt.voterCount; i++ {
				config.AddMember(string(rune('a'+i)), "localhost:5000", Voter)
			}

			quorum := config.Quorum()
			if quorum != tt.wantQuorum {
				t.Errorf("expected quorum %d, got %d", tt.wantQuorum, quorum)
			}
		})
	}
}

func TestConfiguration_GetVotersAndLearners(t *testing.T) {
	config := NewConfiguration()
	config.AddMember("node1", "localhost:5001", Voter)
	config.AddMember("node2", "localhost:5002", Voter)
	config.AddMember("learner1", "localhost:5003", Learner)
	config.AddMember("learner2", "localhost:5004", Learner)

	voters := config.GetVoters()
	if len(voters) != 2 {
		t.Errorf("expected 2 voters, got %d", len(voters))
	}

	learners := config.GetLearners()
	if len(learners) != 2 {
		t.Errorf("expected 2 learners, got %d", len(learners))
	}
}

func TestConfiguration_Clone(t *testing.T) {
	config := NewConfiguration()
	config.AddMember("node1", "localhost:5001", Voter)
	config.Index = 100
	config.Term = 5

	clone := config.Clone()

	// Verify clone is equal
	if clone.Index != config.Index || clone.Term != config.Term {
		t.Error("clone index/term doesn't match original")
	}

	if !clone.Contains("node1") {
		t.Error("clone doesn't contain original members")
	}

	// Verify it's a deep copy
	clone.AddMember("node2", "localhost:5002", Voter)
	if config.Contains("node2") {
		t.Error("modifying clone affected original")
	}
}

func TestConfigurationManager_ProposeChange_AddLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	cm := NewConfigurationManager(members)

	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node4",
		Address: "localhost:5004",
	}

	err := cm.ProposeChange(change)
	if err != nil {
		t.Fatalf("failed to propose change: %v", err)
	}

	if !cm.IsChangeInProgress() {
		t.Error("change should be in progress")
	}

	pending := cm.GetPendingChange()
	if pending == nil || pending.NodeID != "node4" {
		t.Error("pending change not set correctly")
	}

	// Try to propose another change while one is in progress
	change2 := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node5",
		Address: "localhost:5005",
	}
	err = cm.ProposeChange(change2)
	if err == nil {
		t.Error("expected error when proposing change while one is in progress")
	}
}

func TestConfigurationManager_CommitChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	cm := NewConfigurationManager(members)

	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node4",
		Address: "localhost:5004",
	}

	err := cm.ProposeChange(change)
	if err != nil {
		t.Fatalf("failed to propose change: %v", err)
	}

	err = cm.CommitChange(100, 5)
	if err != nil {
		t.Fatalf("failed to commit change: %v", err)
	}

	if cm.IsChangeInProgress() {
		t.Error("change should not be in progress after commit")
	}

	config := cm.GetConfiguration()
	if !config.Contains("node4") {
		t.Error("new member not in configuration after commit")
	}

	if config.IsVoter("node4") {
		t.Error("new member should be a learner")
	}

	if config.Index != 100 || config.Term != 5 {
		t.Error("configuration index/term not updated correctly")
	}
}

func TestConfigurationManager_AbortChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	cm := NewConfigurationManager(members)

	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node2",
		Address: "localhost:5002",
	}

	err := cm.ProposeChange(change)
	if err != nil {
		t.Fatalf("failed to propose change: %v", err)
	}

	cm.AbortChange()

	if cm.IsChangeInProgress() {
		t.Error("change should not be in progress after abort")
	}

	config := cm.GetConfiguration()
	if config.Contains("node2") {
		t.Error("new member should not be in configuration after abort")
	}
}

func TestConfigurationManager_ValidateChange_AddLearner(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	cm := NewConfigurationManager(members)

	// Try to add existing member
	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node1",
		Address: "localhost:5001",
	}
	err := cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when adding existing member")
	}

	// Try to add without address
	change = &ConfigChange{
		Type:   AddLearnerNode,
		NodeID: "node2",
	}
	err = cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when adding learner without address")
	}
}

func TestConfigurationManager_ValidateChange_RemoveNode(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	cm := NewConfigurationManager(members)

	// Try to remove last voter
	change := &ConfigChange{
		Type:   RemoveNode,
		NodeID: "node1",
	}
	err := cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when removing last voter")
	}

	// Try to remove non-existent member
	change = &ConfigChange{
		Type:   RemoveNode,
		NodeID: "node2",
	}
	err = cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when removing non-existent member")
	}
}

func TestConfigurationManager_ValidateChange_PromoteNode(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	cm := NewConfigurationManager(members)

	// Add a learner
	cm.ApplyConfigChange(&ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "learner1",
		Address: "localhost:5002",
	}, 1, 1)

	// Promote the learner
	change := &ConfigChange{
		Type:   PromoteNode,
		NodeID: "learner1",
	}
	err := cm.ProposeChange(change)
	if err != nil {
		t.Errorf("failed to propose promote change: %v", err)
	}

	// Try to promote non-existent member
	change = &ConfigChange{
		Type:   PromoteNode,
		NodeID: "nonexistent",
	}
	err = cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when promoting non-existent member")
	}

	// Try to promote a voter
	change = &ConfigChange{
		Type:   PromoteNode,
		NodeID: "node1",
	}
	err = cm.ProposeChange(change)
	if err == nil {
		t.Error("expected error when promoting a voter")
	}
}

func TestConfigurationManager_ApplyConfigChange(t *testing.T) {
	members := map[string]string{
		"node1": "localhost:5001",
	}
	cm := NewConfigurationManager(members)

	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node2",
		Address: "localhost:5002",
	}

	err := cm.ApplyConfigChange(change, 100, 5)
	if err != nil {
		t.Fatalf("failed to apply config change: %v", err)
	}

	config := cm.GetConfiguration()
	if !config.Contains("node2") {
		t.Error("new member not in configuration")
	}

	if config.Index != 100 || config.Term != 5 {
		t.Error("configuration index/term not set correctly")
	}
}

func TestConfigurationManager_FullWorkflow(t *testing.T) {
	// Start with 3 voters
	members := map[string]string{
		"node1": "localhost:5001",
		"node2": "localhost:5002",
		"node3": "localhost:5003",
	}
	cm := NewConfigurationManager(members)

	config := cm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Errorf("expected 3 voters, got %d", config.VoterCount())
	}

	// Add a learner
	change := &ConfigChange{
		Type:    AddLearnerNode,
		NodeID:  "node4",
		Address: "localhost:5004",
	}
	if err := cm.ProposeChange(change); err != nil {
		t.Fatalf("failed to propose add learner: %v", err)
	}
	if err := cm.CommitChange(1, 1); err != nil {
		t.Fatalf("failed to commit add learner: %v", err)
	}

	config = cm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Errorf("voter count should still be 3, got %d", config.VoterCount())
	}
	if len(config.GetLearners()) != 1 {
		t.Errorf("expected 1 learner, got %d", len(config.GetLearners()))
	}

	// Promote the learner
	change = &ConfigChange{
		Type:   PromoteNode,
		NodeID: "node4",
	}
	if err := cm.ProposeChange(change); err != nil {
		t.Fatalf("failed to propose promote: %v", err)
	}
	if err := cm.CommitChange(2, 1); err != nil {
		t.Fatalf("failed to commit promote: %v", err)
	}

	config = cm.GetConfiguration()
	if config.VoterCount() != 4 {
		t.Errorf("expected 4 voters, got %d", config.VoterCount())
	}
	if len(config.GetLearners()) != 0 {
		t.Errorf("expected 0 learners, got %d", len(config.GetLearners()))
	}

	// Remove a node
	change = &ConfigChange{
		Type:   RemoveNode,
		NodeID: "node1",
	}
	if err := cm.ProposeChange(change); err != nil {
		t.Fatalf("failed to propose remove: %v", err)
	}
	if err := cm.CommitChange(3, 1); err != nil {
		t.Fatalf("failed to commit remove: %v", err)
	}

	config = cm.GetConfiguration()
	if config.VoterCount() != 3 {
		t.Errorf("expected 3 voters after removal, got %d", config.VoterCount())
	}
	if config.Contains("node1") {
		t.Error("node1 should be removed")
	}
}

func TestMemberType_String(t *testing.T) {
	tests := []struct {
		memberType MemberType
		want       string
	}{
		{Voter, "Voter"},
		{Learner, "Learner"},
		{MemberType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.memberType.String()
		if got != tt.want {
			t.Errorf("MemberType.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestConfigChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ConfigChangeType
		want       string
	}{
		{AddLearnerNode, "AddLearnerNode"},
		{PromoteNode, "PromoteNode"},
		{RemoveNode, "RemoveNode"},
		{DemoteNode, "DemoteNode"},
		{ConfigChangeType(99), "Unknown"},
	}

	for _, tt := range tests {
		got := tt.changeType.String()
		if got != tt.want {
			t.Errorf("ConfigChangeType.String() = %v, want %v", got, tt.want)
		}
	}
}
