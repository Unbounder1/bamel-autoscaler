# Server Registration

Servers represent physical bare-metal machines the autoscaler can power on/off. Each Server defines power control and the Kubernetes nodes it can host.

`Server` is cluster-scoped (no namespace) since physical machines are shared infrastructure, similar to `Node` resources.

---

## Server Resource

```yaml
apiVersion: autoscaler.bamel.io/v1alpha1
kind: Server
metadata:
  name: bare-metal-001
spec:
  management:
    powerControl:
      provider: ipmi
      endpoint: "10.0.1.100"
      credentials:
        secretName: "bm-001-ipmi"
        secretNamespace: "autoscaler-system"
  
  nodePool:
    template:
      resources:
        cpu: 16
        memory: "32Gi"
      labels:
        hardware-type: bare-metal
        topology.kubernetes.io/zone: us-east-1a
        node.kubernetes.io/instance-type: "bare-metal-large"
      taints: []
    size:
      min: 0
      max: 2
  
  autoscaling:
    nodeGroup: "bare-metal-compute"
```

---

## Parameter Reference

### `spec.management.powerControl`

Controls how the autoscaler communicates with the server's BMC to power on/off.

| Field | Description |
|-------|-------------|
| `provider` | Protocol for power control: `ipmi` or `redfish` |
| `endpoint` | BMC IP address or hostname |
| `credentials.secretName` | K8s secret containing `username` and `password` keys |
| `credentials.secretNamespace` | Namespace where the secret resides |

### `spec.nodePool.template`

Defines what the nodes from this server look like. Used by the scheduler to match pending pods against available capacity.

| Field | Description |
|-------|-------------|
| `resources.cpu` | Allocatable CPU cores per node |
| `resources.memory` | Allocatable memory per node |
| `labels` | Node labels—used for pod nodeSelector/affinity matching |
| `taints` | Node taints—scheduler checks if pods tolerate these |

### `spec.nodePool.size`

Defines scaling bounds for this server.

| Field | Description |
|-------|-------------|
| `min` | Minimum nodes. `0` allows full power-off when idle |
| `max` | Maximum nodes this physical server can host |

### `spec.autoscaling`

Links the server to autoscaler logic.

| Field | Description |
|-------|-------------|
| `nodeGroup` | Groups servers together for collective scaling decisions |