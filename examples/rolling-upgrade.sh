#!/bin/bash
# rolling-upgrade.sh - Example script for performing a rolling upgrade

set -e

# Configuration
SERVER=${SERVER:-"localhost:7000"}
CHECK_INTERVAL=5

echo "KeyVal Rolling Upgrade Script"
echo "============================="
echo ""

# Get list of all nodes
echo "Fetching cluster members..."
# In production, this would parse JSON from the API
NODES=("node1" "node2" "node3")
echo "Nodes to upgrade: ${NODES[*]}"
echo ""

# Verify cluster is healthy
echo "Checking initial cluster health..."
if ! kvctl -server $SERVER cluster health 2>/dev/null | grep -q "Healthy"; then
    echo "ERROR: Cluster is not healthy. Resolve issues before upgrading."
    exit 1
fi
echo "✓ Cluster is healthy"
echo ""

# Upgrade each node
for NODE in "${NODES[@]}"; do
    echo "========================================"
    echo "Upgrading: $NODE"
    echo "========================================"

    # Get current leader
    LEADER=$(kvctl -server $SERVER cluster leader 2>/dev/null | grep "ID:" | awk '{print $2}')
    echo "Current leader: $LEADER"

    # If this node is the leader, transfer leadership first
    if [ "$LEADER" = "$NODE" ]; then
        echo ""
        echo "Step 1: Transferring leadership from $NODE..."

        # Find a different node to transfer to
        for TARGET in "${NODES[@]}"; do
            if [ "$TARGET" != "$NODE" ]; then
                echo "Transferring to $TARGET..."
                if kvctl -server $SERVER cluster transfer $TARGET 2>/dev/null; then
                    echo "✓ Leadership transferred"
                    sleep $CHECK_INTERVAL
                    break
                else
                    echo "WARNING: Transfer to $TARGET failed, trying next..."
                fi
            fi
        done
    fi

    # Get node address before removal (would query from cluster in production)
    NODE_ADDR=$(kvctl -server $SERVER cluster member $NODE 2>/dev/null | grep "Address:" | awk '{print $2}')
    echo ""
    echo "Step 2: Removing $NODE from cluster..."
    if ! kvctl -server $SERVER cluster remove $NODE 2>/dev/null; then
        echo "ERROR: Failed to remove $NODE"
        exit 1
    fi
    echo "✓ Node removed"

    echo ""
    echo "Step 3: Upgrading $NODE binary..."
    # Placeholder for actual upgrade process
    echo "  - Stopping old version..."
    sleep 1
    echo "  - Installing new version..."
    sleep 1
    echo "  - Starting new version..."
    sleep 1
    echo "✓ Binary upgraded"

    echo ""
    echo "Step 4: Adding $NODE back to cluster..."
    if ! kvctl -server $SERVER cluster add $NODE $NODE_ADDR 2>/dev/null; then
        echo "ERROR: Failed to add $NODE back"
        echo "Manual intervention required!"
        exit 1
    fi
    echo "✓ Node added as learner"

    echo ""
    echo "Step 5: Waiting for $NODE to catch up and promote..."
    ATTEMPTS=0
    MAX_ATTEMPTS=60

    while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
        if kvctl -server $SERVER cluster member $NODE 2>/dev/null | grep -q "Type.*Voter"; then
            break
        fi
        echo -n "."
        sleep $CHECK_INTERVAL
        ATTEMPTS=$((ATTEMPTS + 1))
    done

    echo ""
    if [ $ATTEMPTS -ge $MAX_ATTEMPTS ]; then
        echo "ERROR: $NODE did not rejoin as voter within timeout"
        exit 1
    fi
    echo "✓ Node promoted to voter"

    echo ""
    echo "Step 6: Verifying cluster health..."
    if ! kvctl -server $SERVER cluster health 2>/dev/null | grep -q "Healthy"; then
        echo "WARNING: Cluster health degraded after upgrading $NODE"
        echo "Pausing upgrade. Check cluster status."
        exit 1
    fi
    echo "✓ Cluster still healthy"

    echo ""
    echo "✓ $NODE upgraded successfully!"
    echo ""

    # Wait between upgrades
    if [ "$NODE" != "${NODES[-1]}" ]; then
        echo "Waiting 30 seconds before next upgrade..."
        sleep 30
        echo ""
    fi
done

echo "========================================"
echo "Rolling Upgrade Complete!"
echo "========================================"
echo ""
echo "Final cluster state:"
kvctl -server $SERVER cluster info
echo ""
echo "All nodes upgraded successfully!"
