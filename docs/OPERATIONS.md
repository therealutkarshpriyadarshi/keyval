# KeyVal Operations Guide

This guide provides operational procedures for managing a KeyVal cluster in production.

## Table of Contents

- [Cluster Management](#cluster-management)
- [Adding Nodes](#adding-nodes)
- [Removing Nodes](#removing-nodes)
- [Replacing Nodes](#replacing-nodes)
- [Leadership Transfer](#leadership-transfer)
- [Monitoring](#monitoring)
- [Maintenance Operations](#maintenance-operations)
- [Troubleshooting](#troubleshooting)

## Cluster Management

### Checking Cluster Status

View comprehensive cluster information:

```bash
kvctl cluster info
```

Check cluster health:

```bash
kvctl cluster health
```

List all cluster members:

```bash
kvctl cluster members
```

Get detailed information about a specific member:

```bash
kvctl cluster member node1
```

### Finding the Leader

To find the current leader:

```bash
kvctl cluster leader
```

## Adding Nodes

Adding a new node to the cluster is a multi-phase operation:

### Phase 1: Add as Learner

First, the new node is added as a learner (non-voting member):

```bash
kvctl cluster add node4 localhost:7004
```

The system will:
1. Add the node as a learner
2. Begin replicating the log to the new node
3. Track catch-up progress

### Phase 2: Automatic Catch-Up

The leader automatically replicates log entries to the learner. You can monitor progress:

```bash
kvctl cluster member node4
```

Look for the "Learner Status" section showing:
- Match Index: How many log entries have been replicated
- Caught Up: Whether the learner is ready for promotion

### Phase 3: Automatic Promotion

Once the learner has caught up (within 100 log entries of the leader by default), it is automatically promoted to a voting member.

### Manual Promotion

If you need to manually promote a learner:

```bash
kvctl cluster promote node4
```

**Note:** This will fail if the learner hasn't caught up yet.

### Best Practices for Adding Nodes

1. **Add one node at a time** - Don't add multiple nodes concurrently
2. **Monitor progress** - Check that the learner catches up before promoting
3. **Verify cluster health** - Run `kvctl cluster health` after promotion
4. **Check quorum** - Ensure the new quorum size is acceptable

### Example: Adding a Node to a 3-Node Cluster

```bash
# Initial state: 3 nodes (quorum = 2)
kvctl cluster members

# Add new node
kvctl cluster add node4 localhost:7004

# Monitor catch-up
watch -n 1 'kvctl cluster member node4'

# Wait for automatic promotion
# Final state: 4 nodes (quorum = 3)
kvctl cluster info
```

## Removing Nodes

### Safe Node Removal

To remove a node from the cluster:

```bash
kvctl cluster remove node3
```

### Safety Checks

The system prevents unsafe removals:
- ❌ **Cannot remove the last voting member**
- ❌ **Cannot reduce cluster below minimum size**
- ✅ **Can remove learners at any time**

### Removing the Leader

If you need to remove the current leader:

1. **Transfer leadership first:**
   ```bash
   kvctl cluster transfer node2
   ```

2. **Then remove the old leader:**
   ```bash
   kvctl cluster remove node1
   ```

### Best Practices for Removing Nodes

1. **Transfer leadership** if removing the leader
2. **Verify quorum** - Ensure remaining nodes can form a quorum
3. **Monitor cluster health** after removal
4. **Graceful shutdown** - Let the removed node shut down cleanly

### Example: Removing a Node

```bash
# Check current state
kvctl cluster members

# If removing the leader, transfer first
kvctl cluster transfer node2

# Remove the node
kvctl cluster remove node1

# Verify
kvctl cluster info
```

## Replacing Nodes

Replacing a failed node with a new one is a common operation:

### Automatic Replacement

```bash
kvctl cluster replace node2 node4 localhost:7004
```

This command orchestrates the full replacement:
1. Adds node4 as a learner
2. Waits for node4 to catch up
3. Promotes node4 to voter
4. Removes node2

### Manual Replacement

For more control, you can perform the steps manually:

```bash
# 1. Add new node as learner
kvctl cluster add node4 localhost:7004

# 2. Wait for catch-up
kvctl cluster member node4

# 3. Promote new node
kvctl cluster promote node4

# 4. Remove old node
kvctl cluster remove node2
```

### Best Practices for Replacing Nodes

1. **Don't remove the old node first** - Always add the replacement before removing
2. **Wait for catch-up** - Ensure the new node is caught up before promotion
3. **Monitor cluster size** - Maintain odd number of voters (3, 5, 7)
4. **Rolling replacements** - Replace one node at a time

### Example: Replacing a Failed Node

```bash
# Node2 has failed and needs replacement

# Add replacement node
kvctl cluster add node4 localhost:7004

# Monitor catch-up
watch -n 1 'kvctl cluster member node4'

# Once caught up, it auto-promotes, then remove old node
kvctl cluster remove node2

# Verify final state
kvctl cluster members
```

## Leadership Transfer

Gracefully transfer leadership to another node:

### Basic Transfer

```bash
kvctl cluster transfer node2
```

### When to Transfer Leadership

- **Rolling upgrades** - Transfer leadership before upgrading the current leader
- **Maintenance** - Move leadership before taking the leader offline
- **Load balancing** - Distribute leadership across the cluster
- **Decommissioning** - Transfer before removing a node

### Transfer Process

The transfer happens in phases:
1. **Ensure target is caught up** - Leader waits for target to replicate all entries
2. **Stop new requests** - Leader stops accepting new client requests
3. **Send TimeoutNow** - Leader tells target to start election immediately
4. **Step down** - Current leader becomes follower

### Best Practices

1. **Choose a healthy target** - Ensure target node is online and caught up
2. **Monitor the transfer** - Watch for successful completion
3. **Verify new leader** - Check that target became the leader
4. **Set appropriate timeout** - Use `-timeout` flag if needed

### Example: Transfer During Maintenance

```bash
# Before maintenance on node1 (current leader)

# Transfer leadership to node2
kvctl cluster transfer node2

# Verify transfer
kvctl cluster leader

# Now safe to perform maintenance on node1
```

## Monitoring

### Cluster Health Metrics

Monitor these key metrics:

1. **Leader Presence** - Cluster should always have a leader
2. **Quorum Status** - Majority of nodes should be reachable
3. **Member Count** - Track total nodes and voter count
4. **Replication Lag** - Monitor how far behind followers are

### Health Check Command

```bash
kvctl cluster health
```

Output includes:
- Health status (Healthy/Unhealthy)
- Leader presence
- Quorum availability
- Member health count
- Any issues detected

### Monitoring Learners

For nodes being added as learners:

```bash
kvctl cluster member <node-id>
```

Check:
- **Match Index** - How many entries replicated
- **Bytes Replicated** - Total data replicated
- **Caught Up** - Ready for promotion
- **Last Contact** - When leader last contacted learner

### Example Monitoring Script

```bash
#!/bin/bash
# monitor-cluster.sh

while true; do
  echo "=== Cluster Health ==="
  kvctl cluster health

  echo -e "\n=== Cluster Members ==="
  kvctl cluster members

  echo -e "\n=== Leader ==="
  kvctl cluster leader

  sleep 30
done
```

## Maintenance Operations

### Creating Snapshots

Create a snapshot of the current state:

```bash
kvctl snapshot create
```

### Listing Snapshots

View all available snapshots:

```bash
kvctl snapshot list
```

### Log Compaction

Compact the Raft log to reclaim space:

```bash
kvctl compact
```

### Backup and Restore

Create a full backup:

```bash
kvctl backup /path/to/backup.tar.gz
```

Restore from backup:

```bash
kvctl restore /path/to/backup.tar.gz
```

**⚠️ WARNING:** Restore will overwrite all current data!

### Maintenance Best Practices

1. **Regular snapshots** - Create snapshots daily or weekly
2. **Monitor log size** - Compact when logs grow large
3. **Test restores** - Regularly verify backup restoration
4. **Backup before changes** - Create backup before major operations

## Troubleshooting

### No Leader Elected

**Symptoms:** Cluster health shows "No leader elected"

**Causes:**
- Network partition preventing quorum
- Majority of nodes offline
- Clock skew between nodes

**Resolution:**
1. Check network connectivity between nodes
2. Verify majority of nodes are online
3. Check system clocks are synchronized
4. Review logs for election failures

### Split Brain

**Symptoms:** Multiple nodes think they are leader

**This should never happen** - Raft prevents split-brain

**If it does occur:**
1. Stop all nodes immediately
2. Review logs to identify the issue
3. Restore from last known good backup
4. Contact support/review Raft implementation

### Node Won't Join Cluster

**Symptoms:** New node added but not catching up

**Causes:**
- Network connectivity issues
- Incorrect address configuration
- Leader too busy to replicate

**Resolution:**
1. Verify node is reachable at configured address
2. Check firewall rules allow cluster communication
3. Monitor leader's CPU/network usage
4. Check logs on both leader and new node

### Learner Not Promoting

**Symptoms:** Learner caught up but not promoted

**Causes:**
- Promotion callback not set
- Configuration change already in progress
- Manual promotion required

**Resolution:**
1. Check if another config change is in progress
2. Manually promote: `kvctl cluster promote <node-id>`
3. Verify learner is actually caught up
4. Review system logs

### Quorum Lost

**Symptoms:** Cluster cannot commit new entries

**Causes:**
- Too many nodes offline
- Network partition
- Recent node removal reduced quorum size

**Resolution:**
1. Bring offline nodes back online
2. If nodes permanently lost, may need to use recovery mode
3. Verify current voter count vs nodes needed for quorum
4. Review cluster configuration history

### High Replication Lag

**Symptoms:** Followers far behind leader

**Causes:**
- Network bandwidth limitations
- Slow follower disk I/O
- High write rate overwhelming followers

**Resolution:**
1. Check network bandwidth between nodes
2. Monitor disk I/O on slow followers
3. Consider hardware upgrades for slow nodes
4. Reduce write rate if possible
5. Create snapshot to help followers catch up faster

## Common Operational Patterns

### Rolling Upgrade

To upgrade all nodes without downtime:

```bash
#!/bin/bash
# rolling-upgrade.sh

NODES=("node1" "node2" "node3")

for node in "${NODES[@]}"; do
  echo "Upgrading $node..."

  # Transfer leadership if this is the leader
  leader=$(kvctl cluster leader --json | jq -r '.id')
  if [ "$leader" == "$node" ]; then
    # Find another node to transfer to
    target=$(kvctl cluster members --json | jq -r '.[0].id | select(. != "'$node'")')
    kvctl cluster transfer $target
    sleep 5
  fi

  # Remove node
  kvctl cluster remove $node

  # Upgrade the node (external process)
  upgrade-node $node

  # Add back as learner
  address=$(get-node-address $node)
  kvctl cluster add $node $address

  # Wait for promotion
  while ! kvctl cluster member $node --json | jq -e '.type == "Voter"' > /dev/null; do
    sleep 2
  done

  echo "$node upgraded and promoted"
  sleep 10
done

echo "Rolling upgrade complete!"
```

### Scaling Up (3 → 5 nodes)

```bash
# Add two new nodes

# Add node4
kvctl cluster add node4 localhost:7004
# Wait for auto-promotion

# Add node5
kvctl cluster add node5 localhost:7005
# Wait for auto-promotion

# Verify
kvctl cluster info
# Should show: 5 voters, quorum = 3
```

### Scaling Down (5 → 3 nodes)

```bash
# Remove two nodes

# Remove node5
kvctl cluster remove node5

# Remove node4
kvctl cluster remove node4

# Verify
kvctl cluster info
# Should show: 3 voters, quorum = 2
```

## Safety Guidelines

### ✅ Safe Operations

- Adding nodes one at a time
- Removing non-leader nodes
- Transferring leadership
- Creating snapshots
- Viewing cluster status

### ⚠️ Use Caution

- Removing multiple nodes quickly
- Scaling down to single node
- Force operations
- Concurrent configuration changes

### ❌ Never Do

- Modify data files directly
- Skip the learner phase
- Remove nodes to lose quorum
- Edit configuration outside Raft
- Run multiple clusters on same data

## Performance Tuning

### Catch-Up Threshold

Adjust the catch-up threshold for faster/slower promotion:

```go
// Default: 100 entries
mm := cluster.NewMembershipManager(members, 100)

// Faster promotion (less safe)
mm := cluster.NewMembershipManager(members, 10)

// Slower promotion (more safe)
mm := cluster.NewMembershipManager(members, 1000)
```

### Replication Performance

- **Network bandwidth** - Ensure sufficient bandwidth between nodes
- **Disk I/O** - Fast disks improve replication speed
- **Batch size** - Adjust log entry batch size
- **Snapshots** - Use snapshots for large catch-ups

## Additional Resources

- [Architecture Documentation](ARCHITECTURE.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [API Reference](API.md)
- [Week 9 Summary](WEEK_NINE_SUMMARY.md)

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review logs in `/var/log/keyval/`
3. Open an issue on GitHub
4. Contact support team
