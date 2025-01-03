package v1alpha1

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	hpav2 "k8s.io/api/autoscaling/v2"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func (r *CranePodAutoscaler) Validate() error {
	if !equalTargetRefs(r.Spec.VPA, r.Spec.HPA) {
		return fmt.Errorf("spec.VPA.targetRef does not match spec.HPA.scaleTargetRef: %v", cmp.Diff(r.Spec.VPA.TargetRef, r.Spec.HPA.ScaleTargetRef))
	}
	if r.Spec.HPA.MinReplicas == nil {
		return fmt.Errorf("spec.HPA.minReplicas must be set")
	}
	if r.Spec.Behavior.VPACapacityThresholdPercent < 0 || r.Spec.Behavior.VPACapacityThresholdPercent > 100 {
		return fmt.Errorf("spec.Behavior.vpaCapacityThresholdPercent must be between 0 and 100")
	}
	return nil
}

func equalTargetRefs(vpa vpav1.VerticalPodAutoscalerSpec, hpa hpav2.HorizontalPodAutoscalerSpec) bool {
	return vpa.TargetRef.Name == hpa.ScaleTargetRef.Name &&
		vpa.TargetRef.Kind == hpa.ScaleTargetRef.Kind &&
		vpa.TargetRef.APIVersion == hpa.ScaleTargetRef.APIVersion
}
