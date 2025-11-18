#!/bin/bash
# add-node.sh - Example script to add a new node to the cluster

set -e

# Configuration
NEW_NODE_ID=${1:-"node4"}
NEW_NODE_ADDR=${2:-"localhost:7004"}
SERVER=${3:-"localhost:7000"}
CHECK_INTERVAL=2

echo "Adding new node to KeyVal cluster"
echo "=================================="
echo "Node ID: $NEW_NODE_ID"
echo "Address: $NEW_NODE_ADDR"
echo "Server:  $SERVER"
echo ""

# Check cluster health before starting
echo "Checking cluster health..."
if ! kvctl -server $SERVER cluster health | grep -q "Healthy"; then
    echo "ERROR: Cluster is not healthy. Fix issues before adding nodes."
    exit 1
fi
echo "✓ Cluster is healthy"
echo ""

# Add node as learner
echo "Step 1: Adding $NEW_NODE_ID as learner..."
if ! kvctl -server $SERVER cluster add $NEW_NODE_ID $NEW_NODE_ADDR; then
    echo "ERROR: Failed to add node"
    exit 1
fi
echo "✓ Node added as learner"
echo ""

# Monitor catch-up progress
echo "Step 2: Monitoring catch-up progress..."
echo "This may take several seconds to minutes depending on log size..."
echo ""

CAUGHT_UP=false
ATTEMPTS=0
MAX_ATTEMPTS=60  # 2 minutes max

while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
    # Check if node is caught up (this would query the actual API in production)
    if kvctl -server $SERVER cluster member $NEW_NODE_ID 2>/dev/null | grep -q "Caught Up.*true"; then
        CAUGHT_UP=true
        break
    fi

    echo -n "."
    sleep $CHECK_INTERVAL
    ATTEMPTS=$((ATTEMPTS + 1))
done

echo ""

if [ "$CAUGHT_UP" = false ]; then
    echo "WARNING: Node did not catch up within timeout"
    echo "Check cluster status and node logs"
    exit 1
fi

echo "✓ Node caught up"
echo ""

# Check if auto-promoted
echo "Step 3: Waiting for automatic promotion..."
PROMOTED=false
ATTEMPTS=0

while [ $ATTEMPTS -lt 30 ]; do
    if kvctl -server $SERVER cluster member $NEW_NODE_ID 2>/dev/null | grep -q "Type.*Voter"; then
        PROMOTED=true
        break
    fi

    sleep $CHECK_INTERVAL
    ATTEMPTS=$((ATTEMPTS + 1))
done

if [ "$PROMOTED" = false ]; then
    echo "Node not auto-promoted, promoting manually..."
    if ! kvctl -server $SERVER cluster promote $NEW_NODE_ID; then
        echo "ERROR: Failed to promote node"
        exit 1
    fi
fi

echo "✓ Node promoted to voter"
echo ""

# Verify final state
echo "Step 4: Verifying cluster state..."
kvctl -server $SERVER cluster info
echo ""

echo "✓ Node $NEW_NODE_ID successfully added to cluster!"
echo ""
echo "Next steps:"
echo "  - Monitor cluster health: kvctl -server $SERVER cluster health"
echo "  - Check member status: kvctl -server $SERVER cluster member $NEW_NODE_ID"
echo "  - View all members: kvctl -server $SERVER cluster members"
