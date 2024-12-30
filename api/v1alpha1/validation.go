package v1alpha1

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	hpav2 "k8s.io/api/autoscaling/v2"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func (in *CranePodAutoscaler) Validate() error {
	if !equalTargetRefs(in.Spec.VPA, in.Spec.HPA) {
		return fmt.Errorf("spec.VPA.targetRef does not match spec.HPA.scaleTargetRef: %v", cmp.Diff(in.Spec.VPA.TargetRef, in.Spec.HPA.ScaleTargetRef))
	}
	return nil
}

func equalTargetRefs(vpa vpav1.VerticalPodAutoscalerSpec, hpa hpav2.HorizontalPodAutoscalerSpec) bool {
	return vpa.TargetRef.Name == hpa.ScaleTargetRef.Name &&
		vpa.TargetRef.Kind == hpa.ScaleTargetRef.Kind &&
		vpa.TargetRef.APIVersion == hpa.ScaleTargetRef.APIVersion
}
