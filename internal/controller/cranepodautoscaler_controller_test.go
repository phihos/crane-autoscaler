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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1alpha1 "github.com/phihos/crane-autoscaler/api/v1alpha1"
	autoscaling "k8s.io/api/autoscaling/v1"
	hpav2 "k8s.io/api/autoscaling/v2"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

const testNS = "default"

func newCranePodAutoscaler(name string) *autoscalingv1alpha1.CranePodAutoscaler {
	return &autoscalingv1alpha1.CranePodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNS,
		},
		Spec: autoscalingv1alpha1.CranePodAutoscalerSpec{
			HPA: hpav2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: hpav2.CrossVersionObjectReference{
					Kind:       "Deployment",
					Name:       "my-app",
					APIVersion: "apps/v1",
				},
				MinReplicas: ptr.To[int32](2),
				MaxReplicas: 10,
			},
			VPA: vpav1.VerticalPodAutoscalerSpec{
				TargetRef: &autoscaling.CrossVersionObjectReference{
					Kind:       "Deployment",
					Name:       "my-app",
					APIVersion: "apps/v1",
				},
				UpdatePolicy: &vpav1.PodUpdatePolicy{
					UpdateMode: ptr.To[vpav1.UpdateMode](vpav1.UpdateModeAuto),
				},
			},
			Behavior: autoscalingv1alpha1.CranePodAutoscalerBehavior{
				VPACapacityThresholdPercent: 80,
			},
		},
	}
}

func doReconcile(ctx context.Context, name string) (reconcile.Result, error) {
	r := &CranePodAutoscalerReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
	return r.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Name: name, Namespace: testNS},
	})
}

func setHPAStatus(ctx context.Context, name string, desiredReplicas int32) {
	hpa := &hpav2.HorizontalPodAutoscaler{}
	ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, hpa)).To(Succeed())
	hpa.Status.DesiredReplicas = desiredReplicas
	ExpectWithOffset(1, k8sClient.Status().Update(ctx, hpa)).To(Succeed())
}

func setVPARecommendation(ctx context.Context, name string, recommendations []vpav1.RecommendedContainerResources) {
	vpa := &vpav1.VerticalPodAutoscaler{}
	ExpectWithOffset(1, k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: testNS}, vpa)).To(Succeed())
	vpa.Status.Recommendation = &vpav1.RecommendedPodResources{
		ContainerRecommendations: recommendations,
	}
	ExpectWithOffset(1, k8sClient.Status().Update(ctx, vpa)).To(Succeed())
}

func vpaContainerRecommendation(targetCPU, targetMem string) vpav1.RecommendedContainerResources {
	return vpav1.RecommendedContainerResources{
		ContainerName: "app",
		Target: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(targetCPU),
			corev1.ResourceMemory: resource.MustParse(targetMem),
		},
		UpperBound: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1000m"),
			corev1.ResourceMemory: resource.MustParse("1000Mi"),
		},
	}
}

func cleanup(ctx context.Context, name string) {
	nn := types.NamespacedName{Name: name, Namespace: testNS}

	cpa := &autoscalingv1alpha1.CranePodAutoscaler{}
	if err := k8sClient.Get(ctx, nn, cpa); err == nil {
		_ = k8sClient.Delete(ctx, cpa)
	}
	hpa := &hpav2.HorizontalPodAutoscaler{}
	if err := k8sClient.Get(ctx, nn, hpa); err == nil {
		_ = k8sClient.Delete(ctx, hpa)
	}
	vpa := &vpav1.VerticalPodAutoscaler{}
	if err := k8sClient.Get(ctx, nn, vpa); err == nil {
		_ = k8sClient.Delete(ctx, vpa)
	}
}

var _ = Describe("CranePodAutoscaler Controller", func() {
	ctx := context.Background()

	nn := func(name string) types.NamespacedName {
		return types.NamespacedName{Name: name, Namespace: testNS}
	}

	Context("initial reconciliation", func() {
		const name = "test-initial"

		AfterEach(func() { cleanup(ctx, name) })

		It("creates enabled HPA and disabled VPA", func() {
			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			result, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))

			// Verify HPA is enabled (MaxReplicas matches spec).
			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))
			Expect(*hpa.Spec.MinReplicas).To(Equal(int32(2)))

			// Verify VPA is disabled (UpdateMode=Off).
			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(vpav1.UpdateModeOff))

			// Verify status conditions.
			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			available := meta.FindStatusCondition(cpa.Status.Conditions, "Available")
			Expect(available).NotTo(BeNil())
			Expect(available.Status).To(Equal(metav1.ConditionTrue))

			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("HPA"))

			// Verify owner references on HPA.
			Expect(hpa.OwnerReferences).To(HaveLen(1))
			Expect(hpa.OwnerReferences[0].Name).To(Equal(name))
			Expect(*hpa.OwnerReferences[0].Controller).To(BeTrue())
			Expect(*hpa.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())

			// Verify owner references on VPA.
			Expect(vpa.OwnerReferences).To(HaveLen(1))
			Expect(vpa.OwnerReferences[0].Name).To(Equal(name))
			Expect(*vpa.OwnerReferences[0].Controller).To(BeTrue())
			Expect(*vpa.OwnerReferences[0].BlockOwnerDeletion).To(BeTrue())
		})
	})

	Context("validation", func() {
		It("rejects mismatched target refs", func() {
			const name = "test-mismatch"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			cpa.Spec.VPA.TargetRef.Name = "other-deployment"
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).To(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			available := meta.FindStatusCondition(cpa.Status.Conditions, "Available")
			Expect(available).NotTo(BeNil())
			Expect(available.Status).To(Equal(metav1.ConditionFalse))
			Expect(available.Message).To(ContainSubstring("Validation failed"))
		})

		It("rejects missing minReplicas", func() {
			const name = "test-no-minreplicas"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			cpa.Spec.HPA.MinReplicas = nil
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).To(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			available := meta.FindStatusCondition(cpa.Status.Conditions, "Available")
			Expect(available).NotTo(BeNil())
			Expect(available.Status).To(Equal(metav1.ConditionFalse))
			Expect(available.Message).To(ContainSubstring("minReplicas"))
		})
	})

	Context("scaling decision state machine", func() {
		It("transitions from HPA to VPA when HPA at min replicas and VPA below threshold", func() {
			const name = "test-hpa-to-vpa"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			// First reconcile: creates HPA+VPA, defaults to HPA active.
			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set HPA at min replicas and VPA below threshold (70% utilization).
			setHPAStatus(ctx, name, 2)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("700m", "700Mi"),
			})

			// Second reconcile: should switch to VPA.
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("VPA"))

			// VPA should be enabled (UpdateMode=Auto).
			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(vpav1.UpdateModeAuto))

			// HPA should be disabled (MaxReplicas=MinReplicas=2).
			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(2)))
		})

		It("transitions from VPA to HPA when VPA above threshold", func() {
			const name = "test-vpa-to-hpa"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			// First reconcile: defaults to HPA.
			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set conditions for HPA->VPA transition.
			setHPAStatus(ctx, name, 2)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("700m", "700Mi"),
			})

			// Second reconcile: switches to VPA.
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			Expect(meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision").Reason).To(Equal("VPA"))

			// Now set VPA above threshold (90% utilization).
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("900m", "900Mi"),
			})

			// Third reconcile: should switch back to HPA.
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("HPA"))

			// HPA should be enabled (MaxReplicas=10).
			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))

			// VPA should be disabled (UpdateMode=Off).
			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(vpav1.UpdateModeOff))
		})

		It("stays on HPA when DesiredReplicas > MinReplicas", func() {
			const name = "test-stay-hpa"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// HPA is actively scaling (desired > min) and VPA below threshold.
			setHPAStatus(ctx, name, 5)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "500Mi"),
			})

			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("HPA"))
		})

		It("stays on VPA when utilization remains below threshold", func() {
			const name = "test-stay-vpa"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			// First reconcile: defaults to HPA.
			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Transition to VPA.
			setHPAStatus(ctx, name, 2)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("600m", "600Mi"),
			})
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			Expect(meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision").Reason).To(Equal("VPA"))

			// VPA still below threshold (65% utilization) -- should stay on VPA.
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("650m", "650Mi"),
			})

			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("VPA"))
		})
	})

	Context("spec drift correction", func() {
		It("corrects HPA MaxReplicas drift", func() {
			const name = "test-hpa-drift"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set VPA recommendation and HPA status so re-reconcile doesn't panic on nil recommendation.
			setHPAStatus(ctx, name, 5)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "500Mi"),
			})

			// Manually tamper with HPA MaxReplicas.
			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			hpa.Spec.MaxReplicas = 99
			Expect(k8sClient.Update(ctx, hpa)).To(Succeed())

			// Reconcile should correct it back to 10 (HPA is enabled).
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))
		})

		It("corrects VPA UpdateMode drift", func() {
			const name = "test-vpa-drift"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set VPA recommendation and HPA status so re-reconcile doesn't panic on nil recommendation.
			setHPAStatus(ctx, name, 5)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "500Mi"),
			})

			// VPA should be disabled (Off) since HPA is active.
			// Manually tamper to Auto.
			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			vpa.Spec.UpdatePolicy.UpdateMode = ptr.To[vpav1.UpdateMode](vpav1.UpdateModeAuto)
			Expect(k8sClient.Update(ctx, vpa)).To(Succeed())

			// Reconcile should correct it back to Off.
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(vpav1.UpdateModeOff))
		})
	})

	Context("edge cases", func() {
		It("handles deleted resource gracefully", func() {
			result, err := doReconcile(ctx, "nonexistent-resource")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(reconcile.Result{}))
		})

		It("treats exactly 80% utilization as below threshold (stays on / transitions to VPA)", func() {
			const name = "test-boundary"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set HPA at min replicas, VPA exactly at 80% (800m/1000m).
			setHPAStatus(ctx, name, 2)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("800m", "800Mi"),
			})

			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// 0.8 is NOT > 0.8, so should switch to VPA.
			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("VPA"))
		})

		It("triggers HPA transition when memory exceeds threshold even if CPU is below", func() {
			const name = "test-mem-threshold"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			// First reconcile: HPA active.
			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Transition to VPA first.
			setHPAStatus(ctx, name, 2)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "500Mi"),
			})
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			Expect(meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision").Reason).To(Equal("VPA"))

			// Now set CPU below threshold (50%) but memory above threshold (90%).
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "900Mi"),
			})

			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Should switch to HPA because memory utilization exceeds threshold.
			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())
			decision := meta.FindStatusCondition(cpa.Status.Conditions, "ScalingDecision")
			Expect(decision).NotTo(BeNil())
			Expect(decision.Reason).To(Equal("HPA"))
		})
	})

	Context("owner references", func() {
		It("sets controller owner references on both HPA and VPA", func() {
			const name = "test-ownerrefs"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Re-fetch the CR to get its UID.
			Expect(k8sClient.Get(ctx, nn(name), cpa)).To(Succeed())

			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.OwnerReferences).To(HaveLen(1))
			ownerRef := hpa.OwnerReferences[0]
			Expect(ownerRef.UID).To(Equal(cpa.UID))
			Expect(*ownerRef.Controller).To(BeTrue())
			Expect(*ownerRef.BlockOwnerDeletion).To(BeTrue())

			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(vpa.OwnerReferences).To(HaveLen(1))
			ownerRef = vpa.OwnerReferences[0]
			Expect(ownerRef.UID).To(Equal(cpa.UID))
			Expect(*ownerRef.Controller).To(BeTrue())
			Expect(*ownerRef.BlockOwnerDeletion).To(BeTrue())
		})
	})

	Context("idempotent re-reconciliation", func() {
		It("does not error on repeated reconciles without state change", func() {
			const name = "test-idempotent"
			defer cleanup(ctx, name)

			cpa := newCranePodAutoscaler(name)
			Expect(k8sClient.Create(ctx, cpa)).To(Succeed())

			// First reconcile creates resources.
			_, err := doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// Set VPA recommendation and HPA status so re-reconcile doesn't panic on nil recommendation.
			setHPAStatus(ctx, name, 5)
			setVPARecommendation(ctx, name, []vpav1.RecommendedContainerResources{
				vpaContainerRecommendation("500m", "500Mi"),
			})

			// Second reconcile should be a no-op.
			_, err = doReconcile(ctx, name)
			Expect(err).NotTo(HaveOccurred())

			// HPA still enabled, VPA still disabled.
			hpa := &hpav2.HorizontalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), hpa)).To(Succeed())
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(10)))

			vpa := &vpav1.VerticalPodAutoscaler{}
			Expect(k8sClient.Get(ctx, nn(name), vpa)).To(Succeed())
			Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(vpav1.UpdateModeOff))
		})
	})
})
