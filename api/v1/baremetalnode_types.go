/*
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
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceCapacity struct {
	// Resource Metrics
	CpuCapacity    string `json:"cpuCapacity"`
	MemoryCapacity string `json:"memoryCapacity"`
	DiskCapacity   string `json:"diskCapacity"`

	GpuCapacity string `json:"gpuCapacity"` // CUDA Cores (NVIDIA) or Stream Processors (AMD)

}

// BareMetalNodeSpec defines the desired state of BareMetalNode.
type BareMetalNodeSpec struct {

	// Hardware specifications
	MgmtAddress    string `json:"mgmtAddress"`    // BMC IP address
	MACAddress     string `json:"macAddress"`     // Primary NIC MAC
	PowerInterface string `json:"powerInterface"` // redfish|ipmi|wol
	BMCCredentials string `json:"bmcCredentials"` // Secret name
	BootProfile    string `json:"bootProfile"`    // PXE boot configuration

	// Physical location and categorization
	Rack string `json:"rack,omitempty"`
	Pool string `json:"pool"` // general|gpu|storage|compute

	// Resource specifications
	Resources ResourceCapacity `json:"resources"`

	// Desired state (set by NodePool controller)
	PowerState string `json:"powerState"` // on|off

	// Integration
	InventoryID string `json:"inventoryId,omitempty"`
}

// BareMetalNodeStatus defines the observed state of BareMetalNode.
type BareMetalNodeStatus struct {
	// Current state
	State      string      `json:"state"` // off|powering-on|ready|draining|error
	StateSince metav1.Time `json:"stateSince"`
	LastError  string      `json:"lastError,omitempty"`

	// Power management
	PowerState      string      `json:"powerState"` // on|off|unknown
	LastPowerAction string      `json:"lastPowerAction,omitempty"`
	PowerActionTime metav1.Time `json:"powerActionTime,omitempty"`

	// Kubernetes integration
	KubernetesNode string `json:"kubernetesNode,omitempty"` // Name of corresponding k8s node
	NodeReady      bool   `json:"nodeReady"`

	// Health and diagnostics
	BMCReachable bool        `json:"bmcReachable"`
	BootTime     metav1.Time `json:"bootTime,omitempty"`
	JoinTime     metav1.Time `json:"joinTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BareMetalNode is the Schema for the baremetalnodes API.
type BareMetalNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BareMetalNodeSpec   `json:"spec,omitempty"`
	Status BareMetalNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BareMetalNodeList contains a list of BareMetalNode.
type BareMetalNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BareMetalNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BareMetalNode{}, &BareMetalNodeList{})
}
