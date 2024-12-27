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

package controller

import (
	"context"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1alpha1 "github.com/phihos/crane-autoscaler/api/v1alpha1"
	autoscaling "k8s.io/api/autoscaling/v1"
	hpav1 "k8s.io/api/autoscaling/v1"
)

var _ = Describe("CranePodAutoscaler Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		cranepodautoscaler := &autoscalingv1alpha1.CranePodAutoscaler{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CranePodAutoscaler")
			err := k8sClient.Get(ctx, typeNamespacedName, cranepodautoscaler)
			if err != nil && errors.IsNotFound(err) {
				resource := &autoscalingv1alpha1.CranePodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: autoscalingv1alpha1.CranePodAutoscalerSpec{
						HPA: hpav1.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: hpav1.CrossVersionObjectReference{},
							MaxReplicas:    20,
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
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &autoscalingv1alpha1.CranePodAutoscaler{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CranePodAutoscaler")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CranePodAutoscalerReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
