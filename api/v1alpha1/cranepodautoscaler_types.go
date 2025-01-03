/*
Copyright 2024.

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
	hpav2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

// Important: Run "make" to regenerate code after modifying this file

// CranePodAutoscalerSpec defines the desired state of CranePodAutoscaler
type CranePodAutoscalerSpec struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	HPA hpav2.HorizontalPodAutoscalerSpec `json:"hpa"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	VPA vpav1.VerticalPodAutoscalerSpec `json:"vpa"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Behavior CranePodAutoscalerBehavior `json:"behavior"`
}

type CranePodAutoscalerBehavior struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:ExclusiveMaximum=false
	// Percentage of the VPA target and the upper bound.
	// Exceeding this threshold will cause autoscaling to switch from vertical to horizontal autoscaling.
	// Falling below this threshold will cause autoscaling to switch from horizontal to vertical autoscaling
	// if the HPA scaled down to min replicas.
	// +operator-sdk:csv:customresourcedefinitions:type=behavior
	VPACapacityThresholdPercent int32 `json:"vpaCapacityThresholdPercent,omitempty"`
}

// CranePodAutoscalerStatus defines the observed state of CranePodAutoscaler
type CranePodAutoscalerStatus struct {
	// Represents the observations of a CraneAutoscaler's current state.
	// CraneAutoscaler.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// CraneAutoscaler.status.conditions.status are one of True, False, Unknown.
	// CraneAutoscaler.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// CraneAutoscaler.status.conditions.Message is a human-readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Conditions store the status conditions of the CraneAutoscaler instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:spec
// +kubebuilder:subresource:status

// CranePodAutoscaler is the Schema for the cranepodautoscalers API
type CranePodAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CranePodAutoscalerSpec   `json:"spec,omitempty"`
	Status CranePodAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CranePodAutoscalerList contains a list of CranePodAutoscaler
type CranePodAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CranePodAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CranePodAutoscaler{}, &CranePodAutoscalerList{})
}
