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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	autoscaling "k8s.io/api/autoscaling/v1"
	hpav2 "k8s.io/api/autoscaling/v2"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CranePodAutoscaler Webhook", func() {

	Context("When creating CranePodAutoscaler under Validating Webhook", func() {
		It("Should deny if target refs are different", func() {
			resource := &CranePodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "default",
				},
				Spec: CranePodAutoscalerSpec{
					HPA: hpav2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: hpav2.CrossVersionObjectReference{
							Kind:       "Deployment",
							Name:       "some-deployment",
							APIVersion: "apps/v1",
						},
						MaxReplicas: 20,
					},
					VPA: vpav1.VerticalPodAutoscalerSpec{
						TargetRef: &autoscaling.CrossVersionObjectReference{
							Kind:       "Deployment",
							Name:       "some-other-deployment",
							APIVersion: "apps/v1",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).NotTo(Succeed())

		})

		It("Should admit if all required fields are provided", func() {
			resource := &CranePodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resource",
					Namespace: "default",
				},
				Spec: CranePodAutoscalerSpec{
					HPA: hpav2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: hpav2.CrossVersionObjectReference{
							Kind:       "Deployment",
							Name:       "some-deployment",
							APIVersion: "apps/v1",
						},
						MaxReplicas: 20,
					},
					VPA: vpav1.VerticalPodAutoscalerSpec{
						TargetRef: &autoscaling.CrossVersionObjectReference{
							Kind:       "Deployment",
							Name:       "some-deployment",
							APIVersion: "apps/v1",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})
	})
})
