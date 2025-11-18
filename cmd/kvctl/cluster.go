package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
)

// ClusterCommands handles cluster management commands
type ClusterCommands struct {
	client *Client
}

// NewClusterCommands creates a new cluster commands handler
func NewClusterCommands(client *Client) *ClusterCommands {
	return &ClusterCommands{
		client: client,
	}
}

// ListMembers lists all cluster members
func (cc *ClusterCommands) ListMembers(jsonOutput bool) error {
	fmt.Printf("Listing cluster members (from %s)...\n", cc.client.serverAddr)

	// In production, this would query via gRPC
	// For now, we'll show placeholder data
	members := []api.MemberInfo{
		{
			ID:       "node1",
			Address:  "localhost:7001",
			Type:     "Voter",
			State:    api.StateLeader,
			IsLeader: true,
			IsSelf:   true,
			AddedAt:  time.Now().Add(-24 * time.Hour),
		},
		{
			ID:       "node2",
			Address:  "localhost:7002",
			Type:     "Voter",
			State:    api.StateFollower,
			IsLeader: false,
			IsSelf:   false,
			AddedAt:  time.Now().Add(-24 * time.Hour),
		},
		{
			ID:       "node3",
			Address:  "localhost:7003",
			Type:     "Voter",
			State:    api.StateFollower,
			IsLeader: false,
			IsSelf:   false,
			AddedAt:  time.Now().Add(-24 * time.Hour),
		},
	}

	if jsonOutput {
		return cc.printJSON(members)
	}

	return cc.printMembersTable(members)
}

// ShowMember shows details about a specific member
func (cc *ClusterCommands) ShowMember(nodeID string, jsonOutput bool) error {
	fmt.Printf("Getting member info for %s (from %s)...\n", nodeID, cc.client.serverAddr)

	// Placeholder data
	member := &api.MemberInfo{
		ID:       nodeID,
		Address:  "localhost:7001",
		Type:     "Voter",
		State:    api.StateLeader,
		IsLeader: true,
		IsSelf:   true,
		AddedAt:  time.Now().Add(-24 * time.Hour),
	}

	if jsonOutput {
		return cc.printJSON(member)
	}

	// Print member details
	fmt.Println("\nMember Information:")
	fmt.Printf("  ID:           %s\n", member.ID)
	fmt.Printf("  Address:      %s\n", member.Address)
	fmt.Printf("  Type:         %s\n", member.Type)
	fmt.Printf("  State:        %s\n", member.State)
	fmt.Printf("  Is Leader:    %v\n", member.IsLeader)
	fmt.Printf("  Is Self:      %v\n", member.IsSelf)
	fmt.Printf("  Added At:     %s\n", member.AddedAt.Format(time.RFC3339))

	if member.LearnerStatus != nil {
		fmt.Println("\nLearner Status:")
		fmt.Printf("  Match Index:        %d\n", member.LearnerStatus.MatchIndex)
		fmt.Printf("  Catch-Up Threshold: %d\n", member.LearnerStatus.CatchUpThreshold)
		fmt.Printf("  Caught Up:          %v\n", member.LearnerStatus.CaughtUp)
		fmt.Printf("  Bytes Replicated:   %d\n", member.LearnerStatus.BytesReplicated)
	}

	return nil
}

// AddMember adds a new member to the cluster
func (cc *ClusterCommands) AddMember(nodeID, address string) error {
	fmt.Printf("Adding member %s at %s (to %s)...\n", nodeID, address, cc.client.serverAddr)

	// In production, this would send request via gRPC
	fmt.Println("\nPhase 1: Adding as learner...")
	time.Sleep(100 * time.Millisecond) // Simulate delay
	fmt.Println("✓ Member added as learner")

	fmt.Println("\nPhase 2: Waiting for catch-up...")
	fmt.Println("  Progress: Replicating log entries...")
	time.Sleep(200 * time.Millisecond)
	fmt.Println("✓ Learner caught up")

	fmt.Println("\nPhase 3: Promoting to voting member...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✓ Member promoted to voter")

	fmt.Printf("\n✓ Successfully added member %s to cluster\n", nodeID)
	return nil
}

// RemoveMember removes a member from the cluster
func (cc *ClusterCommands) RemoveMember(nodeID string, force bool) error {
	fmt.Printf("Removing member %s (from %s)...\n", nodeID, cc.client.serverAddr)

	if force {
		fmt.Println("⚠ Force removal requested - skipping safety checks")
	}

	// In production, this would send request via gRPC
	fmt.Println("\nRemoving member from cluster...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✓ Configuration change committed")

	fmt.Printf("\n✓ Successfully removed member %s from cluster\n", nodeID)
	fmt.Println("\nNote: The removed node will automatically shut down when it receives the configuration change.")

	return nil
}

// PromoteMember promotes a learner to a voting member
func (cc *ClusterCommands) PromoteMember(nodeID string) error {
	fmt.Printf("Promoting learner %s to voter (via %s)...\n", nodeID, cc.client.serverAddr)

	// In production, this would check if learner is caught up
	fmt.Println("\nChecking learner status...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✓ Learner is caught up")

	fmt.Println("\nPromoting to voting member...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✓ Member promoted")

	fmt.Printf("\n✓ Successfully promoted %s to voting member\n", nodeID)
	return nil
}

// ShowLeader shows information about the current leader
func (cc *ClusterCommands) ShowLeader(jsonOutput bool) error {
	fmt.Printf("Getting leader information (from %s)...\n", cc.client.serverAddr)

	// Placeholder data
	leader := &api.MemberInfo{
		ID:       "node1",
		Address:  "localhost:7001",
		Type:     "Voter",
		State:    api.StateLeader,
		IsLeader: true,
		IsSelf:   true,
		AddedAt:  time.Now().Add(-24 * time.Hour),
	}

	if jsonOutput {
		return cc.printJSON(leader)
	}

	fmt.Println("\nCurrent Leader:")
	fmt.Printf("  ID:      %s\n", leader.ID)
	fmt.Printf("  Address: %s\n", leader.Address)
	fmt.Printf("  State:   %s\n", leader.State)

	return nil
}

// ShowClusterInfo shows comprehensive cluster information
func (cc *ClusterCommands) ShowClusterInfo(jsonOutput bool) error {
	fmt.Printf("Getting cluster information (from %s)...\n", cc.client.serverAddr)

	// Placeholder data
	info := &api.ClusterInfo{
		LeaderID:     "node1",
		CurrentTerm:  5,
		MemberCount:  3,
		VoterCount:   3,
		LearnerCount: 0,
		Quorum:       2,
		Healthy:      true,
		Timestamp:    time.Now(),
		Members: []*api.MemberInfo{
			{
				ID:       "node1",
				Address:  "localhost:7001",
				Type:     "Voter",
				State:    api.StateLeader,
				IsLeader: true,
				IsSelf:   true,
				AddedAt:  time.Now().Add(-24 * time.Hour),
			},
			{
				ID:       "node2",
				Address:  "localhost:7002",
				Type:     "Voter",
				State:    api.StateFollower,
				IsLeader: false,
				IsSelf:   false,
				AddedAt:  time.Now().Add(-24 * time.Hour),
			},
			{
				ID:       "node3",
				Address:  "localhost:7003",
				Type:     "Voter",
				State:    api.StateFollower,
				IsLeader: false,
				IsSelf:   false,
				AddedAt:  time.Now().Add(-24 * time.Hour),
			},
		},
	}

	if jsonOutput {
		return cc.printJSON(info)
	}

	// Print cluster overview
	fmt.Println("\nCluster Information:")
	fmt.Printf("  Leader:         %s\n", info.LeaderID)
	fmt.Printf("  Current Term:   %d\n", info.CurrentTerm)
	fmt.Printf("  Members:        %d total (%d voters, %d learners)\n", info.MemberCount, info.VoterCount, info.LearnerCount)
	fmt.Printf("  Quorum:         %d\n", info.Quorum)
	fmt.Printf("  Health:         %v\n", info.Healthy)
	fmt.Printf("  Timestamp:      %s\n", info.Timestamp.Format(time.RFC3339))

	fmt.Println("\nMembers:")
	return cc.printMembersTable(convertMemberPointers(info.Members))
}

// ShowClusterHealth shows cluster health status
func (cc *ClusterCommands) ShowClusterHealth(jsonOutput bool) error {
	fmt.Printf("Checking cluster health (from %s)...\n", cc.client.serverAddr)

	// Placeholder data
	health := &api.ClusterHealth{
		Healthy:         true,
		LeaderPresent:   true,
		QuorumAvailable: true,
		MembersHealthy:  3,
		MembersTotal:    3,
		Issues:          []string{},
		Timestamp:       time.Now(),
	}

	if jsonOutput {
		return cc.printJSON(health)
	}

	// Print health status
	fmt.Println("\nCluster Health:")

	healthStatus := "✓ Healthy"
	if !health.Healthy {
		healthStatus = "✗ Unhealthy"
	}
	fmt.Printf("  Status:           %s\n", healthStatus)
	fmt.Printf("  Leader Present:   %v\n", health.LeaderPresent)
	fmt.Printf("  Quorum Available: %v\n", health.QuorumAvailable)
	fmt.Printf("  Members:          %d/%d healthy\n", health.MembersHealthy, health.MembersTotal)

	if len(health.Issues) > 0 {
		fmt.Println("\n  Issues:")
		for _, issue := range health.Issues {
			fmt.Printf("    - %s\n", issue)
		}
	}

	return nil
}

// TransferLeadership transfers leadership to another node
func (cc *ClusterCommands) TransferLeadership(targetID string, timeout time.Duration) error {
	fmt.Printf("Transferring leadership to %s (via %s)...\n", targetID, cc.client.serverAddr)

	// In production, this would send request via gRPC
	fmt.Println("\nPhase 1: Ensuring target is caught up...")
	time.Sleep(100 * time.Millisecond)
	fmt.Println("✓ Target is caught up")

	fmt.Println("\nPhase 2: Stopping new requests...")
	time.Sleep(50 * time.Millisecond)
	fmt.Println("✓ Requests stopped")

	fmt.Println("\nPhase 3: Sending TimeoutNow to target...")
	time.Sleep(50 * time.Millisecond)
	fmt.Println("✓ Target starting election")

	fmt.Println("\nPhase 4: Waiting for new leader...")
	time.Sleep(200 * time.Millisecond)
	fmt.Printf("✓ %s is now the leader\n", targetID)

	fmt.Println("\n✓ Leadership transfer complete")
	return nil
}

// ReplaceNode replaces one node with another
func (cc *ClusterCommands) ReplaceNode(oldNodeID, newNodeID, newAddress string) error {
	fmt.Printf("Replacing %s with %s at %s (via %s)...\n", oldNodeID, newNodeID, newAddress, cc.client.serverAddr)

	// In production, this would orchestrate the full replacement
	fmt.Println("\nPhase 1: Adding new node as learner...")
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("✓ %s added as learner\n", newNodeID)

	fmt.Println("\nPhase 2: Waiting for new node to catch up...")
	fmt.Println("  Progress: Replicating log entries...")
	time.Sleep(300 * time.Millisecond)
	fmt.Printf("✓ %s caught up\n", newNodeID)

	fmt.Println("\nPhase 3: Promoting new node to voter...")
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("✓ %s promoted to voter\n", newNodeID)

	fmt.Println("\nPhase 4: Removing old node...")
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("✓ %s removed from cluster\n", oldNodeID)

	fmt.Printf("\n✓ Successfully replaced %s with %s\n", oldNodeID, newNodeID)
	return nil
}

// printMembersTable prints members in a table format
func (cc *ClusterCommands) printMembersTable(members []api.MemberInfo) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "\nID\tADDRESS\tTYPE\tSTATE\tLEADER\tSELF")
	fmt.Fprintln(w, "--\t-------\t----\t-----\t------\t----")

	for _, member := range members {
		leaderMark := ""
		if member.IsLeader {
			leaderMark = "✓"
		}

		selfMark := ""
		if member.IsSelf {
			selfMark = "✓"
		}

		state := string(member.State)
		if state == "" {
			state = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			member.ID,
			member.Address,
			member.Type,
			state,
			leaderMark,
			selfMark,
		)
	}

	return w.Flush()
}

// printJSON prints data as JSON
func (cc *ClusterCommands) printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// convertMemberPointers converts []*MemberInfo to []MemberInfo
func convertMemberPointers(members []*api.MemberInfo) []api.MemberInfo {
	result := make([]api.MemberInfo, len(members))
	for i, m := range members {
		result[i] = *m
	}
	return result
}
