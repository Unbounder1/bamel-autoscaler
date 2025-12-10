# Server Registration Guide

## Overview

Two ways to register servers with the autoscaler:
1. **Manual Registration** - Explicitly define all node specs
2. **Discovery** - Auto-detect node specs from running K8s cluster

---

## Manual Registration

### Full Specification

Provide all parameters explicitly:
```bash
# Complete manual registration
bamel-autoscaler server register \
  --id bare-metal-001 \
  --cpu-cores 32 \
  --memory-gb 64 \
  --power-watts 200 \
  --bmc-address 10.0.1.100 \
  --bmc-method ipmi \
  --bmc-credentials-secret bm-001-ipmi \
  --bmc-credentials-namespace autoscaler-system \
  --node-prefix bm-001-node \
  --max-nodes 2 \
  --node-cpu 16 \
  --node-memory 32 \
  --labels "hardware-type=bare-metal,zone=us-east-1a" \
  --topology-rack rack-a \
  --topology-datacenter dc-east \
  --autoscaler-enabled \
  --min-nodes 0 \
  --max-nodes 2 \
  --node-group bare-metal-compute
```

### Minimal Manual Registration

Required parameters only (uses defaults):
```bash
# Minimal registration
bamel-autoscaler server register \
  --id bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --bmc-credentials-secret bm-001-ipmi \
  --node-prefix bm-001-node \
  --max-nodes 2
```

**Defaults:**
- `cpu-cores`: Discovered from sum of nodes
- `memory-gb`: Discovered from sum of nodes
- `power-watts`: `0` (will estimate if possible)
- `bmc-method`: `ipmi`
- `bmc-credentials-namespace`: Same namespace as Server resource
- `node-cpu/memory`: Auto-calculated from hardware / max-nodes
- `min-nodes`: `0`
- `autoscaler-enabled`: `true`

### YAML Definition

Equivalent YAML for version control:
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: bare-metal-001
  namespace: autoscaler-system
spec:
  # Optional: inherit defaults from a profile
  profile: standard-compute
  
  # Physical server properties
  hardware:
    cpuCores: 32
    memoryGB: 64
    powerWatts: 200
    location: "rack-a-slot-3"
    tags:
      purchased: "2024-01"
      warranty: "2027-01"
  
  # Failure domain / topology awareness
  topology:
    rack: "rack-a"
    datacenter: "dc-east"
    zone: "us-east-1a"
    powerCircuit: "pdu-3"  # For blast radius awareness
  
  # How to control the physical server
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.100"
      credentials:
        secretName: "bm-001-ipmi"
        namespace: "autoscaler-system"  # Optional, defaults to Server's namespace
      timeoutSeconds: 300
  
  # How to provision Kubernetes nodes from this server
  nodePool:
    # Template for nodes (use for homogeneous pools)
    template:
      resources:
        cpu: 16
        memory: "32Gi"
      labels:
        hardware-type: bare-metal
        topology.kubernetes.io/zone: us-east-1a
        node.kubernetes.io/instance-type: "bare-metal-large"
      taints: []
    
    # OR: Multiple templates for heterogeneous pools
    # templates:
    #   - name: large
    #     count: 2
    #     resources:
    #       cpu: 32
    #       memory: "128Gi"
    #   - name: small
    #     count: 4
    #     resources:
    #       cpu: 8
    #       memory: "32Gi"
    
    # How many nodes this server can host
    size:
      min: 0      # Can scale to zero (power off)
      max: 2      # Physical limit
      desired: 1  # Initial state
    
    # Node naming
    naming:
      prefix: "bm-001-node"
  
  # How this integrates with cluster autoscaler
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-compute"
    scaleDownDelay: 10m
    scaleDownUnneededTime: 10m
```
```bash
kubectl apply -f servers/bare-metal-001.yaml
```

---

## Server Profiles

Reduce boilerplate by defining common configurations as profiles:

```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: ServerProfile
metadata:
  name: standard-compute
  namespace: autoscaler-system
spec:
  # Default node pool settings
  nodePool:
    template:
      labels:
        hardware-type: bare-metal
        environment: production
    size:
      min: 0
  
  # Default autoscaling settings  
  autoscaling:
    enabled: true
    scaleDownDelay: 10m
    scaleDownUnneededTime: 10m
  
  # Default management settings
  management:
    powerControl:
      provider: ipmi
      timeoutSeconds: 300
```

Then reference in Server resources:
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: bare-metal-001
spec:
  profile: standard-compute  # Inherit defaults
  
  # Only specify what differs from profile
  hardware:
    cpuCores: 32
    memoryGB: 64
  
  management:
    powerControl:
      endpoint: "10.0.1.100"
      credentials:
        secretName: "bm-001-ipmi"
  
  nodePool:
    size:
      max: 2
    naming:
      prefix: "bm-001-node"
```

---

## Discovery-Based Registration

### Automatic Node Detection

Register server, auto-discover node specs from cluster:
```bash
# Prerequisites: Nodes must be labeled with server-id
kubectl label node bm-001-node-0 autoscaler.bamel.io/server-id=bare-metal-001
kubectl label node bm-001-node-1 autoscaler.bamel.io/server-id=bare-metal-001

# Optional: Add BMC info as annotations (enables zero-arg discovery)
kubectl annotate node bm-001-node-0 \
  autoscaler.bamel.io/bmc-address=10.0.1.100 \
  autoscaler.bamel.io/bmc-method=ipmi \
  autoscaler.bamel.io/bmc-secret=bm-001-ipmi \
  autoscaler.bamel.io/bmc-secret-namespace=autoscaler-system

# Full discovery (if BMC annotations exist)
bamel-autoscaler server discover bare-metal-001

# Or specify BMC details explicitly
bamel-autoscaler server discover bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --bmc-credentials-secret bm-001-ipmi
```

**What gets discovered:**
- Hardware specs (sum of all node capacities)
- Node count and naming pattern
- Node resources (CPU, memory per node)
- Labels (common labels across nodes)
- Taints (common taints across nodes)
- Current power state
- BMC details (if annotated on nodes)
- Topology (from standard K8s topology labels)

**Generated Server spec:**
```yaml
spec:
  hardware:
    cpuCores: 32        # Discovered from sum of nodes
    memoryGB: 64        # Discovered from sum of nodes
  
  topology:
    zone: "us-east-1a"  # From topology.kubernetes.io/zone label
    rack: "rack-a"      # From autoscaler.bamel.io/rack annotation
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.100"
      credentials:
        secretName: "bm-001-ipmi"
        namespace: "autoscaler-system"
  
  nodePool:
    template:
      resources:
        cpu: 16          # Discovered from node capacity
        memory: "32Gi"   # Discovered from node capacity
      labels:            # Common labels extracted
        hardware-type: bare-metal
        topology.kubernetes.io/zone: us-east-1a
      taints: []         # Common taints extracted
    
    size:
      max: 2             # Discovered from node count
      desired: 2         # Current running nodes
    
    naming:
      prefix: "bm-001-node"  # Detected from node names
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-compute"
```

### Bulk Discovery

Discover multiple servers at once:
```bash
# Scan entire cluster for labeled nodes
bamel-autoscaler server discover-all \
  --label-selector "autoscaler.bamel.io/server-id" \
  --output servers.yaml

# Output:
# Discovered 3 servers:
#   - bare-metal-001 (2 nodes: bm-001-node-0, bm-001-node-1)
#   - bare-metal-002 (4 nodes: bm-002-node-0 to bm-002-node-3)
#   - bare-metal-003 (2 nodes: bm-003-node-0, bm-003-node-1)
# 
# Total hardware capacity:
#   - bare-metal-001: 32 cores, 64GB RAM
#   - bare-metal-002: 64 cores, 128GB RAM
#   - bare-metal-003: 32 cores, 64GB RAM
#
# Generated: servers.yaml
#
# Review and apply:
#   kubectl apply -f servers.yaml
```

### Partial Discovery (Hybrid)

Specify some parameters, discover the rest:
```bash
# Specify hardware and power control, discover node specs
bamel-autoscaler server register \
  --id bare-metal-001 \
  --cpu-cores 32 \
  --memory-gb 64 \
  --bmc-address 10.0.1.100 \
  --bmc-credentials-secret bm-001-ipmi \
  --discover-nodes  # Auto-discover node pool config from K8s
```
```bash
# Or specify node pool, discover hardware
bamel-autoscaler server register \
  --id bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --node-prefix bm-001-node \
  --discover-hardware  # Query nodes to calculate total hardware
```

---

## Registration Workflow

### Pre-Provisioned Servers
```
1. Ensure nodes are in cluster and labeled
   
   kubectl get nodes
   kubectl label node bm-001-node-0 autoscaler.bamel.io/server-id=bare-metal-001
   kubectl label node bm-001-node-1 autoscaler.bamel.io/server-id=bare-metal-001
   
   # Optional: Add BMC annotations for full discovery
   kubectl annotate node bm-001-node-0 \
     autoscaler.bamel.io/bmc-address=10.0.1.100 \
     autoscaler.bamel.io/bmc-secret=bm-001-ipmi

2. Create BMC credentials secret
   
   kubectl create secret generic bm-001-ipmi \
     --from-literal=username=admin \
     --from-literal=password=<password> \
     -n autoscaler-system

3. (Optional) Create a profile for common settings
   
   kubectl apply -f profiles/standard-compute.yaml

4. Register server (choose one):
   
   Option A - Full manual:
   bamel-autoscaler server register \
     --id bare-metal-001 \
     --cpu-cores 32 \
     --memory-gb 64 \
     --bmc-address 10.0.1.100 \
     --node-prefix bm-001-node \
     --max-nodes 2
   
   Option B - Auto-discover (with BMC annotations):
   bamel-autoscaler server discover bare-metal-001
   
   Option C - Auto-discover (specify BMC):
   bamel-autoscaler server discover bare-metal-001 \
     --bmc-address 10.0.1.100 \
     --bmc-credentials-secret bm-001-ipmi
   
   Option D - YAML with profile:
   kubectl apply -f servers/bare-metal-001.yaml

5. Verify registration
   
   bamel-autoscaler server list
   bamel-autoscaler server describe bare-metal-001
   
   # Output shows:
   # Server: bare-metal-001
   # Profile: standard-compute
   # Hardware: 32 cores, 64GB RAM, 200W
   # Topology: dc-east / rack-a / us-east-1a
   # Power State: on
   # Node Pool: 2/2 nodes (bm-001-node-0, bm-001-node-1)
   # Status: Ready

6. Test power control (optional)
   
   bamel-autoscaler server power-status bare-metal-001
   # Output: Status: on, Uptime: 5d 3h
   
   bamel-autoscaler server power-off bare-metal-001 --drain-nodes
   # Drains nodes, then powers off server
   
   bamel-autoscaler server power-on bare-metal-001
   # Powers on, waits for nodes to rejoin

7. Server is now managed by autoscaler
   
   # Autoscaler will:
   # - Scale node pool size between min (0) and max (2)
   # - Power off server when all nodes drained
   # - Power on server when nodes needed
   # - Respect topology for scale-down decisions
   # - Track node state changes in Server.status
```

### Dynamic Provisioning
```
1. Define server template with cloud provider details
   
   kubectl apply -f servers/aws-template.yaml

2. Controller validates template
   
   - Cloud credentials: ✓
   - Instance type exists: ✓
   - Bootstrap script valid: ✓

3. First scale-up triggers provisioning
   
   - Instance created in cloud
   - Bootstraps and joins cluster
   - Node pool automatically updated

4. Subsequent operations use cached specs
```

---

## Discovery Requirements

### Node Labeling

**Required label** for discovery to work:
```bash
kubectl label node <node-name> autoscaler.bamel.io/server-id=<server-id>
```

**Optional annotations** for enhanced discovery:
```bash
# BMC details (enables zero-arg discovery)
kubectl annotate node bm-001-node-0 \
  autoscaler.bamel.io/bmc-address=10.0.1.100 \
  autoscaler.bamel.io/bmc-method=ipmi \
  autoscaler.bamel.io/bmc-secret=bm-001-ipmi \
  autoscaler.bamel.io/bmc-secret-namespace=autoscaler-system

# Topology hints
kubectl annotate node bm-001-node-0 \
  autoscaler.bamel.io/rack=rack-a \
  autoscaler.bamel.io/power-circuit=pdu-3

# Hardware hints (for validation)
kubectl annotate node bm-001-node-0 \
  autoscaler.bamel.io/server-cpu-cores=32 \
  autoscaler.bamel.io/server-memory-gb=64
```

### Discovery Validation

After discovery, controller validates:
```
✓ All nodes have server-id label
✓ Node naming pattern detected (e.g., bm-001-node-*)
✓ Hardware capacity calculated from node sum
✓ Common labels/taints extracted
✓ BMC address reachable
✓ Power control method supported
✓ Credentials valid (test power status query)
✓ Topology labels consistent across nodes
```

**If validation fails:**
```bash
# Check discovery results
bamel-autoscaler server describe bare-metal-001

# Common issues:
# - Nodes not labeled: Add server-id label
# - Inconsistent node names: Use consistent prefix
# - BMC unreachable: Check network/firewall
# - Credentials invalid: Check secret in correct namespace
# - Node specs vary: Use manual registration or heterogeneous templates
# - Topology mismatch: Ensure topology labels match across nodes
```

---

## Heterogeneous Node Pools

For servers that host nodes with different resource configurations:

```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: mixed-server-001
spec:
  hardware:
    cpuCores: 64
    memoryGB: 256
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.100"
      credentials:
        secretName: "mixed-001-ipmi"
  
  nodePool:
    # Use templates (plural) for heterogeneous pools
    templates:
      - name: large
        count: 2
        resources:
          cpu: 24
          memory: "96Gi"
        labels:
          node-size: large
      
      - name: small
        count: 4
        resources:
          cpu: 4
          memory: "16Gi"
        labels:
          node-size: small
    
    size:
      min: 0
      max: 6  # 2 large + 4 small
    
    naming:
      prefix: "mixed-001-node"
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-mixed"
```

**Discovery with heterogeneous nodes:**
```bash
# Discovery detects varying node sizes and generates templates
bamel-autoscaler server discover mixed-server-001 \
  --bmc-address 10.0.1.100 \
  --bmc-credentials-secret mixed-001-ipmi

# Output:
# Detected heterogeneous node pool:
#   - Template "large": 2 nodes (24 CPU, 96Gi each)
#   - Template "small": 4 nodes (4 CPU, 16Gi each)
# 
# Review generated spec before applying.
```

---

## Status Tracking

After registration, the Server status automatically tracks node state:
```yaml
status:
  conditions:
    - type: PoweredOn
      status: "True"
      lastTransitionTime: "2024-12-10T08:00:00Z"
    - type: Ready
      status: "True"
      reason: NodesHealthy
    - type: ProfileApplied
      status: "True"
      reason: "standard-compute"
  
  hardware:
    powerState: on
    uptimeSeconds: 432000  # 5 days
  
  nodePool:
    currentSize: 2    # 2 nodes currently running
    readyNodes: 2     # Both nodes ready
    nodes:
      - name: bm-001-node-0
        uid: "abc-123"
        ready: true
        schedulable: true
        template: default  # Or template name for heterogeneous pools
        
        # Shows what's different from spec
        specDrift:
          labelsAdded:
            custom-app-label: "production"
          taintsAdded:
            - key: node.kubernetes.io/disk-pressure
              effect: NoSchedule
              timeAdded: "2024-12-10T10:00:00Z"
        
        resources:
          allocatable:
            cpu: "15.8"
            memory: "31Gi"
        
        conditions:
          ready: true
          diskPressure: true
          memoryPressure: false
          pidPressure: false
      
      - name: bm-001-node-1
        uid: "def-456"
        ready: true
        schedulable: true
        template: default
        specDrift: null  # No drift from spec
        resources:
          allocatable:
            cpu: "15.8"
            memory: "31Gi"
        conditions:
          ready: true
          diskPressure: false
          memoryPressure: false
          pidPressure: false
  
  lastUpdated: "2024-12-10T10:30:00Z"
```

**Note:** Resource utilization (CPU/memory percentages) is intentionally omitted from status to avoid expensive polling. Query on-demand:
```bash
bamel-autoscaler server utilization bare-metal-001
```

### Monitoring Drift

View nodes that have diverged from spec:
```bash
# Show all servers with drift
bamel-autoscaler server list --show-drift

# Output:
# NAME             POWER    NODES    DRIFT      TOPOLOGY
# bare-metal-001   on       2/2      1 node     dc-east/rack-a
# bare-metal-002   on       4/4      No drift   dc-east/rack-b
# bare-metal-003   off      0/2      -          dc-west/rack-a

# Get detailed drift information
bamel-autoscaler server drift bare-metal-001

# Output:
# Server: bare-metal-001
# 
# Node: bm-001-node-0
#   Labels Added:
#     custom-app-label: "production"
#   Taints Added:
#     - node.kubernetes.io/disk-pressure (NoSchedule)
#       Added: 2024-12-10T10:00:00Z
# 
# Node: bm-001-node-1
#   No drift detected
```

---

## Command Reference

### Registration Commands
```bash
# Manual registration
bamel-autoscaler server register \
  --id <server-id> \
  --cpu-cores <cores> \
  --memory-gb <memory> \
  --bmc-address <ip> \
  --bmc-credentials-secret <secret> \
  [--bmc-credentials-namespace <ns>] \
  --node-prefix <prefix> \
  --max-nodes <count> \
  [--profile <profile-name>] \
  [--node-cpu <cores>] \
  [--node-memory <gb>] \
  [--labels <key=val,key=val>] \
  [--power-watts <watts>] \
  [--min-nodes <count>] \
  [--node-group <group>] \
  [--topology-rack <rack>] \
  [--topology-datacenter <dc>] \
  [--topology-zone <zone>]

# Discovery-based registration (zero-arg if BMC annotated)
bamel-autoscaler server discover <server-id> \
  [--bmc-address <ip>] \
  [--bmc-credentials-secret <secret>] \
  [--profile <profile-name>]

# Bulk discovery
bamel-autoscaler server discover-all \
  --label-selector <label> \
  [--output <file>] \
  [--profile <profile-name>]

# Hybrid registration
bamel-autoscaler server register \
  --id <server-id> \
  --bmc-address <ip> \
  --discover-hardware \
  --discover-nodes
```

### Profile Commands
```bash
# List profiles
bamel-autoscaler profile list

# Show profile details
bamel-autoscaler profile describe <profile-name>

# Create profile from YAML
kubectl apply -f profiles/standard-compute.yaml

# Show servers using a profile
bamel-autoscaler profile servers <profile-name>
```

### Query Commands
```bash
# List all registered servers
bamel-autoscaler server list
bamel-autoscaler server list --show-drift
bamel-autoscaler server list --topology  # Group by datacenter/rack

# Show server details
bamel-autoscaler server describe <server-id>

# Show node pool status
bamel-autoscaler server nodes <server-id>

# Show drift from spec
bamel-autoscaler server drift <server-id>

# Show resource utilization (on-demand query)
bamel-autoscaler server utilization <server-id>
```

### Power Control Commands
```bash
# Check power status
bamel-autoscaler server power-status <server-id>

# Power operations
bamel-autoscaler server power-on <server-id>
bamel-autoscaler server power-off <server-id> [--drain-nodes]
bamel-autoscaler server power-cycle <server-id>

# Test power control
bamel-autoscaler server test-power <server-id>
```

### Validation Commands
```bash
# Validate server configuration
bamel-autoscaler server validate <server-id>

# Validate all servers
bamel-autoscaler server validate-all

# Check node specs against K8s reality
bamel-autoscaler server validate <server-id> --check-nodes

# Show differences
bamel-autoscaler server validate <server-id> --show-diff

# Validate profile
bamel-autoscaler profile validate <profile-name>
```

### Update Commands
```bash
# Re-discover node specs
bamel-autoscaler server update <server-id> --discover-nodes

# Re-discover hardware specs
bamel-autoscaler server update <server-id> --discover-hardware

# Update power control settings
bamel-autoscaler server update <server-id> \
  --bmc-address <new-ip> \
  --bmc-credentials-secret <new-secret>

# Update node pool template
bamel-autoscaler server update <server-id> \
  --node-cpu <cores> \
  --node-memory <gb> \
  --add-label "key=value" \
  --remove-label "old-key"

# Update autoscaling settings
bamel-autoscaler server update <server-id> \
  --min-nodes <count> \
  --max-nodes <count> \
  --node-group <group>

# Change profile
bamel-autoscaler server update <server-id> --profile <new-profile>

# Reconcile drift (align nodes with spec)
bamel-autoscaler server reconcile <server-id>
```

### Topology Commands
```bash
# List servers by topology
bamel-autoscaler server list --topology

# Output:
# DATACENTER   RACK      SERVERS   TOTAL NODES   POWER
# dc-east      rack-a    3         8/10          2 on, 1 off
# dc-east      rack-b    2         6/6           2 on
# dc-west      rack-a    4         12/16         3 on, 1 off

# Show servers in specific topology
bamel-autoscaler server list --datacenter dc-east --rack rack-a

# Check topology blast radius before scale-down
bamel-autoscaler server check-topology --datacenter dc-east
# Output: Warning: Scaling down bare-metal-003 would leave rack-a with 0 nodes
```

---

## Best Practices

### 1. Start with Discovery
- Label nodes first with server-id
- Add BMC annotations for zero-arg discovery
- Use discovery to generate initial Server resource
- Review generated YAML before applying
- Commit to version control

### 2. Use Profiles for Consistency
- Define profiles for common configurations
- Reduces boilerplate and ensures consistency
- Makes fleet-wide changes easier
- Override only what differs per-server

### 3. Use Consistent Naming
- Node prefix should be unique per server (e.g., `bm-001-node`)
- Allows discovery to detect node pool automatically
- Makes it easy to identify which nodes belong to which server

### 4. Define Topology
- Always set topology fields (datacenter, rack, zone)
- Enables smart scale-down decisions
- Prevents emptying entire racks/datacenters
- Useful for blast radius awareness

### 5. Validate Before Production
- Test power control on each server: `bamel-autoscaler server test-power <id>`
- Verify node specs match reality: `bamel-autoscaler server validate <id>`
- Do trial power-off/power-on cycle with `--drain-nodes`
- Ensure nodes rejoin cluster correctly after power cycle

### 6. Monitor Drift
- Check drift regularly: `bamel-autoscaler server list --show-drift`
- Decide if drift should be:
  - **Accepted**: Update spec to match reality
  - **Corrected**: Reconcile nodes to match spec
- Use drift info for capacity planning

### 7. Keep Configs in Sync
- Store Server and Profile YAMLs in git
- Re-run discovery after hardware changes
- Validate periodically: `bamel-autoscaler server validate-all`
- Document why drift exists if accepting it

### 8. Use Secrets for Credentials
```bash
# Create BMC credentials secret
kubectl create secret generic bm-001-ipmi \
  --from-literal=username=admin \
  --from-literal=password=<password> \
  -n autoscaler-system

# Reference in server config
--bmc-credentials-secret bm-001-ipmi
--bmc-credentials-namespace autoscaler-system
```

### 9. Set Appropriate Limits
```yaml
nodePool:
  size:
    min: 0   # Allow scale to zero for cost savings
    max: 2   # Don't exceed physical capacity
```

---

## Troubleshooting

### Discovery Fails

**No nodes found:**
```bash
# Check labels
kubectl get nodes --show-labels | grep server-id

# Add missing labels
kubectl label node <node> autoscaler.bamel.io/server-id=<server-id>
```

**BMC details not discovered:**
```bash
# Check annotations
kubectl get node <node> -o jsonpath='{.metadata.annotations}' | jq

# Add BMC annotations
kubectl annotate node <node> \
  autoscaler.bamel.io/bmc-address=10.0.1.100 \
  autoscaler.bamel.io/bmc-secret=bm-001-ipmi
```

**Inconsistent node naming:**
```bash
# Discovery expects consistent prefix
# Bad:  k8s-node-001, worker-002
# Good: bm-001-node-0, bm-001-node-1

# Check node names
kubectl get nodes -l autoscaler.bamel.io/server-id=bare-metal-001

# If inconsistent, use manual registration
```

**Hardware capacity mismatch:**
```bash
# Discovery sums all node capacities
# If nodes report different capacities, spec may be inaccurate

# Verify node capacities match
kubectl get nodes -o custom-columns=NAME:.metadata.name,CPU:.status.capacity.cpu,MEM:.status.capacity.memory

# Use manual registration or heterogeneous templates if nodes vary
```

### Registration Validation Fails

**BMC unreachable:**
```bash
# Test connectivity
ping <bmc-address>

# Test IPMI manually
ipmitool -I lanplus -H <bmc-address> -U admin -P <password> power status
```

**Credentials invalid:**
```bash
# Verify secret exists in correct namespace
kubectl get secret <secret-name> -n <namespace>

# Check secret contents
kubectl get secret <secret-name> -n <namespace> -o jsonpath='{.data.username}' | base64 -d
```

**Node count exceeds max:**
```bash
# More nodes running than spec.nodePool.size.max

# Option 1: Increase max
bamel-autoscaler server update <server-id> --max-nodes <new-max>

# Option 2: Remove extra nodes from server-id label
kubectl label node <node> autoscaler.bamel.io/server-id-
```

**Profile not found:**
```bash
# Check profile exists
bamel-autoscaler profile list

# Check profile is in same namespace
kubectl get serverprofile <profile-name> -n autoscaler-system
```

### Drift Detection

**Unexpected labels/taints on nodes:**
```bash
# View drift
bamel-autoscaler server drift <server-id>

# Option 1: Accept drift (update spec to match)
bamel-autoscaler server update <server-id> \
  --add-label "new-label=value" \
  --add-taint "new-taint=value:NoSchedule"

# Option 2: Reconcile nodes to match spec
bamel-autoscaler server reconcile <server-id>
# Warning: This removes drift by updating nodes
```

**Status not updating:**
```bash
# Check controller logs
kubectl logs -n autoscaler-system -l app=bamel-autoscaler-controller

# Force reconciliation
kubectl annotate server <server-id> autoscaler.bamel.io/reconcile="$(date +%s)"

# Verify watch is working
kubectl get events --field-selector involvedObject.kind=Server
```

### Power Control Issues

**Server won't power off:**
```bash
# Check if nodes are drained
kubectl get nodes -l autoscaler.bamel.io/server-id=<server-id>

# Drain manually if needed
kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data

# Then retry power-off
bamel-autoscaler server power-off <server-id>
```

**Server won't power on:**
```bash
# Check BMC status
ipmitool -I lanplus -H <bmc-address> -U admin -P <password> chassis status

# Check controller can reach BMC
kubectl exec -n autoscaler-system <controller-pod> -- ping <bmc-address>

# Check for recent power control errors
bamel-autoscaler server describe <server-id> | grep -A 10 "Recent Events"
```

---

## Examples

### Example 1: Small Bare Metal Server
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: small-server-01
spec:
  profile: standard-compute
  
  hardware:
    cpuCores: 16
    memoryGB: 32
    powerWatts: 150
  
  topology:
    datacenter: dc-east
    rack: rack-a
    zone: us-east-1a
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.10"
      credentials:
        secretName: "small-01-ipmi"
  
  nodePool:
    template:
      resources:
        cpu: 16
        memory: "32Gi"
      labels:
        hardware-type: bare-metal
        size: small
    
    size:
      min: 0
      max: 1
      desired: 1
    
    naming:
      prefix: "small-01-node"
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-small"
```

### Example 2: Large Multi-Node Server
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: large-server-01
spec:
  profile: standard-compute
  
  hardware:
    cpuCores: 128
    memoryGB: 512
    powerWatts: 800
    location: "datacenter-a-rack-05"
  
  topology:
    datacenter: dc-east
    rack: rack-05
    zone: us-east-1a
    powerCircuit: pdu-3
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.50"
      credentials:
        secretName: "large-01-ipmi"
      timeoutSeconds: 600
  
  nodePool:
    template:
      resources:
        cpu: 32
        memory: "128Gi"
      labels:
        hardware-type: bare-metal
        size: large
        gpu: "false"
      taints:
        - key: workload-type
          value: compute-intensive
          effect: NoSchedule
    
    size:
      min: 1      # Always keep at least one node
      max: 4      # Can host 4 nodes
      desired: 2  # Start with 2
    
    naming:
      prefix: "large-01-node"
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-large"
    scaleDownDelay: 30m
    scaleDownUnneededTime: 15m
```

### Example 3: GPU Server
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: gpu-server-01
spec:
  hardware:
    cpuCores: 64
    memoryGB: 256
    powerWatts: 1200
    tags:
      gpuModel: "NVIDIA A100"
      gpuCount: "8"
  
  topology:
    datacenter: dc-west
    rack: gpu-rack-01
    zone: us-west-2a
    powerCircuit: pdu-gpu-1
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.100"
      credentials:
        secretName: "gpu-01-ipmi"
  
  nodePool:
    template:
      resources:
        cpu: 32
        memory: "128Gi"
      labels:
        hardware-type: bare-metal
        accelerator: nvidia-a100
        gpu-count: "4"
      taints:
        - key: nvidia.com/gpu
          value: "true"
          effect: NoSchedule
    
    size:
      min: 0
      max: 2
      desired: 0  # Expensive, start with 0
    
    naming:
      prefix: "gpu-01-node"
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-gpu"
    scaleDownDelay: 5m  # Scale down quickly when idle
```

### Example 4: Heterogeneous Server
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: mixed-server-01
spec:
  hardware:
    cpuCores: 64
    memoryGB: 256
  
  topology:
    datacenter: dc-east
    rack: rack-mixed-01
  
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.200"
      credentials:
        secretName: "mixed-01-ipmi"
  
  nodePool:
    templates:
      - name: large
        count: 2
        resources:
          cpu: 24
          memory: "96Gi"
        labels:
          node-size: large
          workload-class: compute
      
      - name: small
        count: 4
        resources:
          cpu: 4
          memory: "16Gi"
        labels:
          node-size: small
          workload-class: utility
    
    size:
      min: 0
      max: 6
    
    naming:
      prefix: "mixed-01-node"
  
  autoscaling:
    enabled: true
    nodeGroup: "bare-metal-mixed"
```

### Example 5: ServerProfile
```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: ServerProfile
metadata:
  name: standard-compute
  namespace: autoscaler-system
spec:
  nodePool:
    template:
      labels:
        hardware-type: bare-metal
        environment: production
        managed-by: bamel-autoscaler
    size:
      min: 0
  
  autoscaling:
    enabled: true
    scaleDownDelay: 10m
    scaleDownUnneededTime: 10m
  
  management:
    powerControl:
      provider: ipmi
      timeoutSeconds: 300
```