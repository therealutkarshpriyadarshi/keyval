# kvctl - KeyVal CLI Tool

Command-line interface for interacting with KeyVal cluster.

## Commands

```bash
# Key-value operations
kvctl put <key> <value>
kvctl get <key>
kvctl delete <key>

# Cluster management
kvctl status
kvctl members list
kvctl members add <node-id> <address>
kvctl members remove <node-id>

# Maintenance
kvctl snapshot
kvctl compact
```

## Status

Phase 5: Planned
