# Scheduler

## Purpose

Select minimum-power servers to satisfy workload resource requirements.

**Constraint:** Must power on entire servers (all nodes come online together).

## Input Parameters

### 1. Workload Profile
```yaml
name: string
category: string

base_weights:
  cpu: float
  memory: float
  network: float

custom_metrics:
  - name: string
    weight: float

label_requirements: [string]      # Hard constraints (must match)
label_preferences:                # Soft preferences (bonus points)
  - label: string
    weight: float
```

### 2. Resource Requirements
```
required_cpu: float
required_memory: float
required_disk: float
```

### 3. Available Servers
```
server:
  id: string
  power_consumption: float
  nodes: [                        # Could be VMs or bare metal nodes
    {
      id: string
      cpu: float
      memory: float
      labels: map[string]string
      taints: [...]
    }
  ]
```

**Note:** `nodes` represents K8s nodes. Can be:
- Multiple VMs per server (virtualized)
- Single node per server (bare metal)
- Mix of both

## Algorithm

```
1. Filter Phase
   - For each server: filter nodes by label_requirements & taints
   - Aggregate resources from eligible nodes only
   - Keep servers that meet total resource requirements

2. Score Phase
   - Calculate: base_weights × resources + custom_metrics × weights + label_preferences
   - Normalize by power consumption

3. Select Phase
   - Return server with best score/power ratio
```

## Output

```
selected_servers: [server_id]
estimated_power: float
available_resources: {cpu, memory, disk}
```

## Key Behaviors

- **Node filtering**: Only eligible nodes contribute to server capacity
- **Server-level activation**: Powering any node = power entire server
- **Power optimization**: Minimize total watts while meeting requirements
- **Profile-driven**: All scoring controlled by workload profile
- **Topology agnostic**: Works with VMs, bare metal, or mixed environments

## Deployment Modes

**Bare Metal:** 1 server = 1 node
```
Server-A (200W) → node-001 (32 CPU, 64GB)
```

**Virtualized:** 1 server = N nodes
```
Server-A (200W) → node-001 (16 CPU, 32GB)
                → node-002 (16 CPU, 32GB)
```

**Hybrid:** Mixed topology
```
Server-A (200W) → node-001 (32 CPU, 64GB)          [bare metal]
Server-B (150W) → node-002 (8 CPU, 16GB)           [VM]
                → node-003 (8 CPU, 16GB)           [VM]
```

## Future Parameters

- `topology.allow_colocation: bool` - Allow multiple workloads per server
- `topology.avoid_colocation_with: [string]` - Anti-affinity rules
- `max_servers: int` - Multi-server bin-packing limit
- `cost_weight: float` - Balance power vs monetary cost