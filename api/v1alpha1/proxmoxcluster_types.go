/*
Copyright 2023 IONOS Cloud.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	ipamicv1 "sigs.k8s.io/cluster-api-ipam-provider-in-cluster/api/v1alpha2"
)

const (
	// ProxmoxClusterKind the ProxmoxCluster kind.
	ProxmoxClusterKind = "ProxmoxCluster"
	// ClusterFinalizer allows cleaning up resources associated with
	// ProxmoxCluster before removing it from the apiserver.
	ClusterFinalizer = "proxmoxcluster.infrastructure.cluster.x-k8s.io"
)

// ProxmoxClusterSpec defines the desired state of ProxmoxCluster.
type ProxmoxClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`

	// AllowedNodes specifies all Proxmox nodes which will be considered
	// for operations. This implies that VMs can be cloned on different nodes from
	// the node which holds the VM template.
	// +optional
	AllowedNodes []string `json:"allowedNodes,omitempty"`

	// IPv4Config contains information about available IPV4 address pools and the gateway.
	// this can be combined with ipv6Config in order to enable dual stack.
	// either IPv4Config or IPv6Config must be provided.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.addresses.size() > 0",message="IPv4Config addresses must be provided"
	IPv4Config *ipamicv1.InClusterIPPoolSpec `json:"ipv4Config,omitempty"`

	// IPv6Config contains information about available IPV6 address pools and the gateway.
	// this can be combined with ipv4Config in order to enable dual stack.
	// either IPv4Config or IPv6Config must be provided.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.addresses.size() > 0",message="IPv6Config addresses must be provided"
	IPv6Config *ipamicv1.InClusterIPPoolSpec `json:"ipv6Config,omitempty"`

	// DNSServers contains information about nameservers used by machines network-config.
	// +kubebuilder:validation:MinItems=1
	DNSServers []string `json:"dnsServers"`
}

// ProxmoxClusterStatus defines the observed state of ProxmoxCluster.
type ProxmoxClusterStatus struct {
	// Ready indicates that the cluster is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// InClusterIPPoolRef is the reference to the created in cluster ip pool
	// +optional
	InClusterIPPoolRef []corev1.LocalObjectReference `json:"inClusterIpPoolRef,omitempty"`

	// NodeLocations keeps track of which nodes have been selected
	// for different machines.
	// +optional
	NodeLocations *NodeLocations `json:"nodeLocations,omitempty"`

	// Conditions defines current service state of the ProxmoxCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// NodeLocations holds information about the deployment state of
// control plane and worker nodes in Proxmox.
type NodeLocations struct {
	// ControlPlane contains all deployed control plane nodes
	// +optional
	ControlPlane []NodeLocation `json:"controlPlane,omitempty"`

	// Workers contains all deployed worker nodes
	// +optional
	Workers []NodeLocation `json:"workers,omitempty"`
}

// NodeLocation holds information about a single VM
// in Proxmox.
type NodeLocation struct {
	// Machine is the reference of the proxmoxmachine
	Machine corev1.LocalObjectReference `json:"machine"`

	// Node is the Proxmox node
	Node string `json:"node"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=proxmoxclusters,scope=Namespaced,categories=cluster-api,singular=proxmoxcluster
//+kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels['cluster\\.x-k8s\\.io/cluster-name']",description="Cluster"
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready"
//+kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint",description="API Endpoint"

// ProxmoxCluster is the Schema for the proxmoxclusters API.
type ProxmoxCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self.ipv4Config != null || self.ipv6Config != null",message="at least one ip config must be set, either ipv4Config or ipv6Config"
	Spec   ProxmoxClusterSpec   `json:"spec,omitempty"`
	Status ProxmoxClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProxmoxClusterList contains a list of ProxmoxCluster.
type ProxmoxClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProxmoxCluster `json:"items"`
}

// GetConditions returns the observations of the operational state of the ProxmoxCluster resource.
func (c *ProxmoxCluster) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the underlying service state of the ProxmoxCluster to the predescribed clusterv1.Conditions.
func (c *ProxmoxCluster) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

// SetInClusterIPPoolRef will set the reference to the provided InClusterIPPool.
// If nil was provided, the status field will be cleared.
func (c *ProxmoxCluster) SetInClusterIPPoolRef(pool *ipamicv1.InClusterIPPool) {
	if pool == nil || pool.GetName() == "" {
		c.Status.InClusterIPPoolRef = nil
		return
	}

	if c.Status.InClusterIPPoolRef == nil {
		c.Status.InClusterIPPoolRef = []corev1.LocalObjectReference{
			{Name: pool.GetName()},
		}
	}

	found := false
	for _, ref := range c.Status.InClusterIPPoolRef {
		if ref.Name == pool.GetName() {
			found = true
		}
	}
	if !found {
		c.Status.InClusterIPPoolRef = append(c.Status.InClusterIPPoolRef, corev1.LocalObjectReference{Name: pool.GetName()})
	}
}

// AddNodeLocation will add a node location to either the control plane or worker
// node locations based on the `isControlPlane` parameter.
func (c *ProxmoxCluster) AddNodeLocation(loc NodeLocation, isControlPlane bool) {
	if c.Status.NodeLocations == nil {
		c.Status.NodeLocations = new(NodeLocations)
	}

	if !c.HasMachine(loc.Machine.Name, isControlPlane) {
		c.addNodeLocation(loc, isControlPlane)
	}
}

// RemoveNodeLocation removes a node location from the status.
func (c *ProxmoxCluster) RemoveNodeLocation(machineName string, isControlPlane bool) {
	nodeLocations := c.Status.NodeLocations

	if nodeLocations == nil {
		return
	}

	if !c.HasMachine(machineName, isControlPlane) {
		return
	}

	if isControlPlane {
		for i, v := range nodeLocations.ControlPlane {
			if v.Machine.Name == machineName {
				nodeLocations.ControlPlane = append(nodeLocations.ControlPlane[:i], nodeLocations.ControlPlane[i+1:]...)
			}
		}
		return
	}

	for i, v := range nodeLocations.Workers {
		if v.Machine.Name == machineName {
			nodeLocations.Workers = append(nodeLocations.Workers[:i], nodeLocations.Workers[i+1:]...)
		}
	}
}

// UpdateNodeLocation will update the node location based on the provided machine name.
// If the node location does not exist, it will be added.
//
// The function returns true if the value was added or updated, otherwise false.
func (c *ProxmoxCluster) UpdateNodeLocation(machineName, node string, isControlPlane bool) bool {
	if !c.HasMachine(machineName, isControlPlane) {
		loc := NodeLocation{
			Node:    node,
			Machine: corev1.LocalObjectReference{Name: machineName},
		}
		c.AddNodeLocation(loc, isControlPlane)
		return true
	}

	locations := c.Status.NodeLocations.Workers
	if isControlPlane {
		locations = c.Status.NodeLocations.ControlPlane
	}

	for i, loc := range locations {
		if loc.Machine.Name == machineName {
			if loc.Node != node {
				locations[i].Node = node
				return true
			}

			return false
		}
	}

	return false
}

// HasMachine returns if true if a machine was found on any node.
func (c *ProxmoxCluster) HasMachine(machineName string, isControlPlane bool) bool {
	return c.GetNode(machineName, isControlPlane) != ""
}

// GetNode tries to return the Proxmox node for the provided machine name.
func (c *ProxmoxCluster) GetNode(machineName string, isControlPlane bool) string {
	if c.Status.NodeLocations == nil {
		return ""
	}

	if isControlPlane {
		for _, cpl := range c.Status.NodeLocations.ControlPlane {
			if cpl.Machine.Name == machineName {
				return cpl.Node
			}
		}
	} else {
		for _, wloc := range c.Status.NodeLocations.Workers {
			if wloc.Machine.Name == machineName {
				return wloc.Node
			}
		}
	}

	return ""
}

func (c *ProxmoxCluster) addNodeLocation(loc NodeLocation, isControlPlane bool) {
	if isControlPlane {
		c.Status.NodeLocations.ControlPlane = append(c.Status.NodeLocations.ControlPlane, loc)
		return
	}

	c.Status.NodeLocations.Workers = append(c.Status.NodeLocations.Workers, loc)
}

func init() {
	SchemeBuilder.Register(&ProxmoxCluster{}, &ProxmoxClusterList{})
}
