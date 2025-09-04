# bamel-autoscaler

# IN PROGRESS, NOT FUNCTIONAL

A bare-metal Kubernetes autoscaler that dynamically manages physical server resources based on workload demands. Unlike cloud-based autoscalers that provision virtual instances, bamel-autoscaler controls physical hardware through BMC interfaces to power on/off servers as needed.

## Overview

The bamel-autoscaler consists of several key components working together to provide intelligent bare-metal resource management:

- **NodePool Controller**: Manages groups of similar bare-metal nodes and makes scaling decisions
- **BareMetalNode Controller**: Reconciles individual node states and manages power operations
- **etcd Inventory System**: Centralized storage for node hardware specs, states, and metadata
- **Power Management**: Controls server power states via BMC interfaces (Redfish/IPMI/WoL)
- **Metrics Integration**: Uses Prometheus metrics to make scaling decisions
- **Resource Tracking**: Monitors CPU, memory, GPU, storage, and custom resources

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │ NodePool         │    │  Physical Nodes │
│   API Server    │◄──►│ Controller       │    │   (BMC/IPMI)    │
└─────────────────┘    └──────────┬───────┘    └─────────────────┘
         │                       │                       ▲
         │              ┌────────▼────────┐              │
         │              │ BareMetalNode   │              │
         └──────────────►│ Controller      │──────────────┘
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │  etcd Inventory │
                        │  (Node States)  │
                        └────────┬────────┘
                                 │
                                 ▼
                        ┌──────────────────┐
                        │   Prometheus     │
                        │    Metrics       │
                        └──────────────────┘
```

## Components

### NodePool

The NodePool represents a collection of bare-metal nodes with similar hardware specifications that can be scaled together.

```golang
type NodePoolStatus struct {
    ReadyNodes         int                `json:"readyNodes"`         // Nodes powered on and joined to cluster
    PendingNodes       int                `json:"pendingNodes"`       // Nodes being provisioned
    DesiredNodes       int                `json:"desiredNodes"`       // Target number of active nodes
    ScalingNodes       []ScalingNode      `json:"scalingNodes,omitempty"` // Nodes currently scaling
    AvailableResources ResourceCapacity   `json:"availableResources"` // Total resources of ready nodes
    ActiveResources    ResourceCapacity   `json:"activeResources"`    // Currently allocated resources
    PendingResources   ResourceCapacity   `json:"pendingResources"`   // Resources being provisioned
}

type ResourceCapacity struct {
    CPU     resource.Quantity `json:"cpu"`     // millicores (e.g., 16000m = 16 cores)
    Memory  resource.Quantity `json:"memory"`  // bytes (e.g., 64Gi)
    GPU     int               `json:"gpu"`     // count of GPU devices
    Storage resource.Quantity `json:"storage"` // bytes of local storage
    
    // Custom resources (e.g., FPGAs, specialized accelerators)
    CustomResources map[string]resource.Quantity `json:"customResources,omitempty"`
}

type ScalingNode struct {
    NodeName        string           `json:"nodeName"`      // Kubernetes node name
    Action          string           `json:"action"`        // "scale-up" or "scale-down"
    ActionSince     metav1.Time      `json:"actionSince"`   // When scaling action started
    State           string           `json:"state"`         // "pending"|"in-progress"|"completed"|"failed"
    Resources       ResourceCapacity `json:"resources"`     // Resources this node provides
    Reason          string           `json:"reason"`        // Explanation for scaling decision
}
```

### NodePool Reconciliation Loop

The NodePool controller focuses on high-level scaling decisions and delegates individual node management to the BareMetalNode controller:

1. **Resource Analysis**: Calculate the ratio of `ActiveResources` to `AvailableResources` to determine current utilization
2. **Scaling Decision**: 
   - **Scale Up**: If utilization > high threshold (default 70%) and pending pods exist
   - **Scale Down**: If utilization < low threshold (default 30%) for sustained period
3. **Node Selection**: Choose BareMetalNodes for scaling based on:
   - Hardware specifications matching workload requirements
   - Rack diversity for fault tolerance  
   - Current node states from etcd inventory
4. **Scaling Request**: Update the `DesiredNodes` count and add nodes to `ScalingNodes` list
5. **State Monitoring**: Monitor BareMetalNode states through etcd to track scaling progress

### BareMetalNode Controller

The BareMetalNode controller is responsible for managing individual physical servers and maintains a separate lifecycle from Kubernetes nodes:

```golang
type BareMetalNode struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   BareMetalNodeSpec   `json:"spec"`
    Status BareMetalNodeStatus `json:"status"`
}

type BareMetalNodeSpec struct {
    // Hardware specifications
    MgmtAddress     string            `json:"mgmtAddress"`     // BMC IP address
    MACAddress      string            `json:"macAddress"`      // Primary NIC MAC
    PowerInterface  string            `json:"powerInterface"`  // redfish|ipmi|wol
    BMCCredentials  string            `json:"bmcCredentials"`  // Secret name
    BootProfile     string            `json:"bootProfile"`     // PXE boot configuration
    
    // Physical location and categorization
    Rack            string            `json:"rack,omitempty"`
    Pool            string            `json:"pool"`            // general|gpu|storage|compute
    
    // Resource specifications
    Resources       ResourceCapacity  `json:"resources"`
    
    // Desired state (set by NodePool controller)
    PowerState      string            `json:"powerState"`      // on|off
    
    // Integration
    InventoryID     string            `json:"inventoryId,omitempty"`
}

type BareMetalNodeStatus struct {
    // Current state
    State           string            `json:"state"`           // off|powering-on|ready|draining|error
    StateSince      metav1.Time       `json:"stateSince"`
    LastError       string            `json:"lastError,omitempty"`
    
    // Power management
    PowerState      string            `json:"powerState"`      // on|off|unknown
    LastPowerAction string            `json:"lastPowerAction,omitempty"`
    PowerActionTime metav1.Time       `json:"powerActionTime,omitempty"`
    
    // Kubernetes integration
    KubernetesNode  string            `json:"kubernetesNode,omitempty"` // Name of corresponding k8s node
    NodeReady       bool              `json:"nodeReady"`
    
    // Health and diagnostics
    BMCReachable    bool              `json:"bmcReachable"`
    BootTime        metav1.Time       `json:"bootTime,omitempty"`
    JoinTime        metav1.Time       `json:"joinTime,omitempty"`
}
```

#### BareMetalNode Reconciliation Loop

1. **State Synchronization**: Read current state from etcd and compare with desired state
2. **Power Management**: Execute power operations (on/off) via BMC when needed
3. **Boot Monitoring**: Track server boot progress and PXE boot status
4. **Kubernetes Integration**: Monitor for corresponding Kubernetes node creation/deletion
5. **Health Checks**: Verify BMC connectivity and server responsiveness
6. **State Updates**: Write current state back to etcd for NodePool controller consumption

### etcd Inventory System

The etcd server acts as the central source of truth for all bare-metal node information, decoupled from Kubernetes. This separation allows for better reliability and enables management of nodes that aren't currently part of the cluster.    

#### etcd Schema and Usage

**Key Structure:**
```
/baremetal/
├── nodes/
│   ├── {node-id}/
│   │   ├── spec          # BareMetalNodeSpec (hardware, config)
│   │   ├── status        # BareMetalNodeStatus (current state)
│   │   ├── metrics       # Node-specific metrics cache
│   │   └── events        # State change history
│   └── ...
├── pools/
│   ├── {pool-name}/
│   │   ├── members       # List of node IDs in this pool
│   │   └── config        # Pool-specific configuration
│   └── ...
└── global/
    ├── config            # Global autoscaler settings
    └── stats             # Cluster-wide statistics
```

#### Benefits of etcd-based Inventory

- **Decoupled from Kubernetes**: Node inventory persists even if Kubernetes cluster is down
- **Atomic Operations**: Ensure consistent state changes across multiple nodes
- **Watch Capabilities**: Real-time notifications of state changes
- **Historical Data**: Track node state changes over time
- **High Availability**: etcd clustering provides redundancy
- **External Integration**: Other tools can easily query node inventory
- **Faster Queries**: Direct etcd access is faster than Kubernetes API for bulk operations

#### Integration Points

**NodePool Controller** → **etcd**:
- Reads pool membership and node states
- Updates desired states for scaling operations
- Queries resource availability across pools

**BareMetalNode Controller** → **etcd**:
- Watches for spec changes (desired state)
- Updates status with current state
- Records power operation results

**External Tools** → **etcd**:
- DCIM integration for inventory synchronization
- Monitoring tools for health dashboards
- CLI tools for manual node management
- Backup systems for disaster recovery

This architecture enables true separation of concerns: the NodePool controller focuses on scaling logic, while the BareMetalNode controller handles the complexities of physical hardware management, with etcd serving as the reliable communication layer between them.

### Prometheus Metrics Integration

The autoscaler uses Prometheus metrics to make intelligent scaling decisions beyond simple CPU/memory thresholds:

```go
import (
    "github.com/prometheus/client_golang/api/prometheus/v1"
    "github.com/prometheus/common/model"
)
```

#### Key Metrics Monitored

- **Resource Utilization**: `node_cpu_utilization`, `node_memory_utilization`
- **Pod Scheduling**: `pending_pods_total`, `unschedulable_pods_total`
- **Hardware Health**: `node_temperature`, `node_power_consumption`
- **Workload Patterns**: `pod_creation_rate`, `job_completion_rate`
- **Custom Metrics**: Application-specific metrics for specialized workloads

#### Scaling Triggers

```go
type ScalingTrigger struct {
    MetricQuery   string  `yaml:"metricQuery"`   // PromQL query
    Threshold     float64 `yaml:"threshold"`     // Trigger threshold
    Duration      string  `yaml:"duration"`      // How long threshold must be exceeded
    ScaleDirection string `yaml:"scaleDirection"` // "up" or "down"
}
```