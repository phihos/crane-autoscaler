package v1alpha1

import (
	hpav2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpa_types "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

func (r *CranePodAutoscaler) GenerateEnabledVPA() *vpav1.VerticalPodAutoscaler {
	vpaSpec := r.Spec.VPA.DeepCopy()
	return &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: *vpaSpec,
	}
}

func (r *CranePodAutoscaler) GenerateDisabledVPA() *vpav1.VerticalPodAutoscaler {
	vpaSpec := r.Spec.VPA.DeepCopy()
	updateModeOff := vpa_types.UpdateModeOff
	vpaSpec.UpdatePolicy.UpdateMode = &updateModeOff
	return &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: *vpaSpec,
	}
}

func (r *CranePodAutoscaler) GenerateEnabledHPA() *hpav2.HorizontalPodAutoscaler {
	hpaSpec := r.Spec.HPA.DeepCopy()
	return &hpav2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: *hpaSpec,
	}
}

func (r *CranePodAutoscaler) GenerateDisabledHPA() *hpav2.HorizontalPodAutoscaler {
	hpaSpec := r.Spec.HPA.DeepCopy()
	var minReplicas int32
	if hpaSpec.MinReplicas == nil {
		// This case should never happen as validation fails when not setting MinReplicas.
		// See file "validation.go".
		minReplicas = 1
	} else {
		minReplicas = *hpaSpec.MinReplicas
	}

	hpaSpec.MaxReplicas = minReplicas
	return &hpav2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: *hpaSpec,
	}
}
