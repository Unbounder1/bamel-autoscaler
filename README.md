# bamel-autoscaler
A bare metal autoscaler

## General skeleton

Bare metal autoscaler of kind BareMetalAutoscaler

## Architecture Components

### Custom Resource Definition (API schema, what fields custom resources will have):
- Spec: 
- Status:

### Manager (HA setup functionality)

### Machine Inventory Management
- How is the pool of available machines defined?
  - Defined as a configmap in the schema in the scheduler README.md

- Labels/selectors to match machines to autoscaler instances?

## Detection & Monitoring

### Event Triggers (eBPF-based detection):
- ebpf program to determine whether or not the packets coming in from an interface are coming from a certain pod/node, and deal with taints somehow to calculate feasibility
  - using some kind of layer 4 or 7 whichever one is plausible to determine the desired pod to find the node taint
- Kubelet checks for: cpu, memory, storage, storage ephemeral
- Node phase: pending, running, terminated

### Integration: eBPF → Reconcile
- How does eBPF data trigger a reconcile?
- Is eBPF running in-cluster or on nodes?
- How does this integrate with Kubernetes watch/informer pattern?

## Controller Logic

### Watches (Kubernetes Resources)
- Primary watch:
- Secondary watches:

### Reconciliation Loop (Reconcile() function):

#### 1. Fetch Current State
- 

#### 2. Event Trigger Processing
- Reconciliation loop event triggers when a ebpf program determines either network, cpu, or memory pressure increasing, based on some formula
- Event trigger updates some overall statistics, based on the rate of change in available resource

#### 3. Decision Logic (Scale Up/Down)
- Based on if the rate is positive or negative, determine which map of machines to use
- Selecting category based on pressure trigger
- Based on some score like 1x useful pressure and 0.25x non-useful pressure, determine the best machine that is currently not on to be scheduled to turn on
- Closest to ideal score machine gets selected and removed or added based on sign

#### 4. Safety Checks
- Min/max node counts:
- Cooldown periods:
- Rate limiting:
- Cluster-wide resource thresholds:

#### 5. Scale-Down Safety
- Pod eviction/draining:
- PodDisruptionBudgets:
- Critical workload checks:

### State Tracking
- Machine lifecycle states (off → powering-on → joining → ready):
- Prevent duplicate power operations:

## Actuation

### Power Management
- How do you actually power machines on/off? (IPMI, Redfish, BMC API?)
- Credential/endpoint management:
- Power operation failure handling:

### Node Lifecycle After Power-On
- How does node join cluster?
- Pre-provisioning? Auto-registration?
- NotReady state handling:
- Timeout for machines that won't join:

## Status & Observability

### Status Subresource Updates
- Current capacity:
- Pending machines:
- Last scale event:
- Conditions:

### Metrics
-

### Events
-

### Logs
-

## Cleanup & Safety

### Finalizers
- What happens when BareMetalAutoscaler is deleted?
- Power down managed machines or leave running?

### Error Recovery
- Machine won't power on:
- Machine powers on but never joins cluster:
- Network partitions between controller and BMC:

## Requeue Strategy
- When to requeue immediately:
- When to use RequeueAfter:
- Error handling:

Based on a map 

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/bamel-autoscaler:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/bamel-autoscaler:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/bamel-autoscaler:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/bamel-autoscaler/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

