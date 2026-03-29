/*
Copyright 2026.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
	// Size is the size of the memcached deployment
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	Size int32 `json:"size"`

	// Image is the memcached container image
	// +kubebuilder:default="memcached:1.6.9-alpine"
	// +optional
	Image string `json:"image,omitempty"`

	// ContainerPort is the port the memcached container listens on
	// +kubebuilder:default=11211
	// +optional
	ContainerPort int32 `json:"containerPort,omitempty"`
}

type MemcachedStatus struct {
	// Nodes are the names of the memcached pods
	// +optional
	Nodes []string `json:"nodes,omitempty"`
	// Conditions represent the latest available observations of an object's state
	// +operator-sdk:csv:customresourcedefinitions.type=status
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,2,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Size",type="integer",JSONPath=".spec.size",description="The number of memcached pods"
// +kubebuilder:printcolumne:name="Nodes",type="string",JSONPath=".status.nodes",description="The current memcached nodes"

// Memcached is the Schema for the memcacheds API
type Memcached struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +required
	Spec MemcachedSpec `json:"spec,omitempty"`
	// +optional
	Status MemcachedStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MemcachedList contains a list of Memcached
type MemcachedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Memcached `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Memcached{}, &MemcachedList{})
}
