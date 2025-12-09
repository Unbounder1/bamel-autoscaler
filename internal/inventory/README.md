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
  --provisioning-mode pre-provisioned \
  --power-consumption 200.0 \
  --bmc-address 10.0.1.100 \
  --bmc-method ipmi \
  --bmc-interface lanplus \
  --bmc-username admin \
  --bmc-password-secret ipmi-secret \
  --nodes k8s-node-001,k8s-node-002 \
  --cpu 16,16 \
  --memory 32,32 \
  --labels "node-type=compute,zone=us-east-1a"
```

### Minimal Manual Registration

Required parameters only (uses defaults):

```bash
# Minimal registration
bamel-autoscaler server register \
  --id bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --nodes k8s-node-001,k8s-node-002
```

**Defaults:**
- `provisioning-mode`: `pre-provisioned`
- `bmc-method`: `ipmi`
- `bmc-interface`: `lanplus`
- `power-consumption`: `0` (will estimate if possible)
- Node specs: Discovered from K8s if nodes exist

### YAML Definition

Equivalent YAML for version control:

```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: bare-metal-001
spec:
  provisioning_mode: pre-provisioned
  power_consumption: 200.0
  
  power_control:
    method: ipmi
    bmc_address: "10.0.1.100"
    interface: lanplus
    username: admin
    password_secret: ipmi-secret
    timeout: 300
  
  node_specs:
    - name_template: "k8s-node-001"
      cpu: 16
      memory: 32
      labels:
        node-type: compute
        zone: us-east-1a
    
    - name_template: "k8s-node-002"
      cpu: 16
      memory: 32
      labels:
        node-type: compute
        zone: us-east-1a
```

```bash
kubectl apply -f servers/bare-metal-001.yaml
```

---

## Discovery-Based Registration

### Automatic Node Detection

Register server, auto-discover node specs from cluster:

```bash
# Prerequisites: Nodes must be labeled
kubectl label node k8s-node-001 autoscaler.bamel.io/server-id=bare-metal-001
kubectl label node k8s-node-002 autoscaler.bamel.io/server-id=bare-metal-001

# Discover and register
bamel-autoscaler server discover bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --bmc-password-secret ipmi-secret
```

**What gets discovered:**
- Node names (`k8s-node-001`, `k8s-node-002`)
- CPU capacity (from `node.status.capacity.cpu`)
- Memory capacity (from `node.status.capacity.memory`)
- Labels (from `node.metadata.labels`)
- Taints (from `node.spec.taints`)

### Bulk Discovery

Discover multiple servers at once:

```bash
# Scan entire cluster for labeled nodes
bamel-autoscaler server discover-all \
  --label-selector "autoscaler.bamel.io/server-id" \
  --output servers.yaml

# Output:
# Discovered 3 servers:
#   - bare-metal-001 (2 nodes)
#   - bare-metal-002 (4 nodes)
#   - bare-metal-003 (2 nodes)
# 
# Generated: servers.yaml
#
# Review and apply:
#   kubectl apply -f servers.yaml
```

### Partial Discovery (Hybrid)

Specify some parameters, discover the rest:

```bash
# Specify power control, discover node specs
bamel-autoscaler server register \
  --id bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --discover-nodes  # Auto-fill node specs from K8s
```

```bash
# Or discover power consumption only
bamel-autoscaler server register \
  --id bare-metal-001 \
  --bmc-address 10.0.1.100 \
  --nodes k8s-node-001,k8s-node-002 \
  --discover-power-consumption  # Query IPMI for actual power usage
```

---

## Registration Workflow

### Pre-Provisioned Servers

```
1. Ensure nodes are in cluster and labeled
   
   kubectl get nodes
   kubectl label node k8s-node-001 autoscaler.bamel.io/server-id=bare-metal-001

2. Register server (choose one):
   
   Option A - Full manual:
   bamel-autoscaler server register \
     --id bare-metal-001 \
     --bmc-address 10.0.1.100 \
     --nodes k8s-node-001,k8s-node-002 \
     --cpu 16,16 \
     --memory 32,32
   
   Option B - Auto-discover:
   bamel-autoscaler server discover bare-metal-001 \
     --bmc-address 10.0.1.100

3. Verify registration
   
   bamel-autoscaler server list
   bamel-autoscaler server describe bare-metal-001

4. Test power control (optional)
   
   bamel-autoscaler server power-status bare-metal-001
   bamel-autoscaler server power-off bare-metal-001
   bamel-autoscaler server power-on bare-metal-001

5. Server is now managed by autoscaler
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
   - Node specs automatically updated

4. Subsequent operations use cached specs
```

---

## Discovery Requirements

### Node Labeling

**Required label** for discovery to work:

```bash
kubectl label node <node-name> autoscaler.bamel.io/server-id=<server-id>
```

**Optional labels** for enhanced discovery:

```bash
# BMC address hint
kubectl label node k8s-node-001 autoscaler.bamel.io/bmc-address=10.0.1.100

# Power method hint
kubectl label node k8s-node-001 autoscaler.bamel.io/power-method=ipmi

# Power consumption (if known)
kubectl label node k8s-node-001 autoscaler.bamel.io/power-watts=100
```

### Discovery Validation

After discovery, controller validates:

```
✓ All nodes have server-id label
✓ Node specs extracted successfully
✓ BMC address reachable
✓ Power control method supported
✓ Credentials valid (test power status query)
```

**If validation fails:**
```bash
# Check discovery results
bamel-autoscaler server describe bare-metal-001

# Common issues:
# - Nodes not labeled: Add server-id label
# - BMC unreachable: Check network/firewall
# - Credentials invalid: Update secret
# - Node specs incomplete: Use manual registration
```

---

## Command Reference

### Registration Commands

```bash
# Manual registration
bamel-autoscaler server register \
  --id <server-id> \
  --bmc-address <ip> \
  --nodes <node1,node2> \
  [--cpu <cpu1,cpu2>] \
  [--memory <mem1,mem2>] \
  [--labels <key=val,key=val>] \
  [--power-consumption <watts>]

# Discovery-based registration
bamel-autoscaler server discover <server-id> \
  --bmc-address <ip> \
  [--bmc-password-secret <secret>]

# Bulk discovery
bamel-autoscaler server discover-all \
  --label-selector <label> \
  [--output <file>]

# Hybrid registration
bamel-autoscaler server register \
  --id <server-id> \
  --bmc-address <ip> \
  --discover-nodes
```

### Validation Commands

```bash
# List all registered servers
bamel-autoscaler server list

# Show server details
bamel-autoscaler server describe <server-id>

# Test power control
bamel-autoscaler server power-status <server-id>

# Validate node specs against K8s
bamel-autoscaler server validate <server-id>
```

### Update Commands

```bash
# Update node specs (re-discover)
bamel-autoscaler server update <server-id> --discover-nodes

# Update power control settings
bamel-autoscaler server update <server-id> \
  --bmc-address <new-ip> \
  --bmc-password-secret <new-secret>

# Update labels
bamel-autoscaler server update <server-id> \
  --add-label "key=value" \
  --remove-label "old-key"
```

---

## Best Practices

### 1. Start with Discovery
- Label nodes first
- Use discovery to generate initial config
- Review generated YAML before applying
- Commit to version control

### 2. Validate Before Production
- Test power control on each server
- Verify node specs match reality
- Do trial power-off/power-on cycle
- Ensure nodes rejoin cluster correctly

### 3. Keep Configs in Sync
- Store server YAMLs in git
- Re-run discovery after hardware changes
- Validate periodically: `bamel-autoscaler server validate-all`

### 4. Use Secrets for Credentials
```bash
# Create BMC credentials secret
kubectl create secret generic ipmi-bare-metal-001 \
  --from-literal=username=admin \
  --from-literal=password=<password>

# Reference in server config
--bmc-password-secret ipmi-bare-metal-001
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

**Node specs incomplete:**
```bash
# Verify node has capacity info
kubectl get node <node> -o yaml | grep capacity

# If missing, node may not be Ready
kubectl get nodes
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
# Verify secret exists
kubectl get secret <secret-name>

# Check secret contents
kubectl get secret <secret-name> -o yaml
```

### Drift Detection

Nodes changed but server config outdated:

```bash
# Re-discover to sync
bamel-autoscaler server update <server-id> --discover-nodes

# Compare current vs registered
bamel-autoscaler server validate <server-id> --show-diff
```