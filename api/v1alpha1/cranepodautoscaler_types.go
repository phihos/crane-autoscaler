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
	hpav1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

// CranePodAutoscalerSpec defines the desired state of CranePodAutoscaler
type CranePodAutoscalerSpec struct {
	HPA hpav1.HorizontalPodAutoscalerSpec `json:"hpa"`
	VPA vpav1.VerticalPodAutoscalerSpec   `json:"vpa"`
}

// CranePodAutoscalerStatus defines the observed state of CranePodAutoscaler
type CranePodAutoscalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
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
