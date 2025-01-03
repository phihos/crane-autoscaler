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
	"fmt"

	"github.com/google/go-cmp/cmp"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	autoscalingv1alpha1 "github.com/phihos/crane-autoscaler/api/v1alpha1"
	hpav2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Definitions to manage status conditions
const (
	refVPA = "VPA"
	refHPA = "HPA"
	// typeAvailableCraneAutoscaler represents the status of the Deployment reconciliation
	typeAvailableCraneAutoscaler = "Available"
	// typeDegradedCraneAutoscaler represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	// typeDegradedCraneAutoscaler = "Degraded"
	typeScalingDecisionCraneAutoscaler = "ScalingDecision"
)

// CranePodAutoscalerReconciler reconciles a CranePodAutoscaler object
type CranePodAutoscalerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=autoscaling.phihos.github.io,resources=cranepodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.phihos.github.io,resources=cranepodautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling.phihos.github.io,resources=cranepodautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *CranePodAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	craneAutoscaler := &autoscalingv1alpha1.CranePodAutoscaler{}
	err := r.Get(ctx, req.NamespacedName, craneAutoscaler)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			logger.Info("cranepodautoscaler resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get cranepodautoscaler")
		return ctrl.Result{}, err
	}

	if len(craneAutoscaler.Status.Conditions) == 0 {
		meta.SetStatusCondition(&craneAutoscaler.Status.Conditions, metav1.Condition{Type: typeAvailableCraneAutoscaler, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err := r.Status().Update(ctx, craneAutoscaler); err != nil {
			logger.Error(err, "Failed to update cranepodautoscaler status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the cranepodautoscaler Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster, and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, craneAutoscaler); err != nil {
			logger.Error(err, "Failed to re-fetch cranepodautoscaler")
			return ctrl.Result{}, err
		}
	}

	if err := craneAutoscaler.Validate(); err != nil {
		meta.SetStatusCondition(&craneAutoscaler.Status.Conditions, metav1.Condition{Type: typeAvailableCraneAutoscaler,
			Status: metav1.ConditionFalse, Reason: "Reconciling",
			Message: fmt.Sprintf("Validation failed: %s", err)})
		if err := r.Status().Update(ctx, craneAutoscaler); err != nil {
			logger.Error(err, "Failed to update cranepodautoscaler status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Get or create VPA.
	vpaCreated, vpa, err := r.getOrCreateVPA(ctx, craneAutoscaler)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get or create HPA.
	hpaCreated, hpa, err := r.getOrCreateHPA(ctx, craneAutoscaler)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Got VPA and HPA", refVPA, vpa.Name, refHPA, hpa.Name)

	// Now we get to the core logic: We now decide which autoscaler to activate.
	// The other autoscaler will be deactivated.
	var activeAutoscaler string
	var passiveAutoscaler string
	lastScalingDecisionCondition := meta.FindStatusCondition(craneAutoscaler.Status.Conditions, typeScalingDecisionCraneAutoscaler)
	if vpaCreated || hpaCreated {
		// Special case: One or more autoscalers were just created.
		//               When that happens we initialize the scaling decision with HPA
		//               as this is the safer option in terms of availability.
		activeAutoscaler = refHPA
		passiveAutoscaler = refVPA
	} else if lastScalingDecisionCondition == nil {
		// Special case: Autoscalers already exist, but scaling decision has not been recorded to CRD status.
		//               We default to "HPA" as this is the safer option in terms of availability.
		//               This should not happen.
		activeAutoscaler = refHPA
		passiveAutoscaler = refVPA
	} else {
		// Usual case: VPA and HPA both already exist.
		// 			   Now our action depends on the current scaling mode.
		currentlyActiveAutoscaler := lastScalingDecisionCondition.Reason
		containerName, biggestUtilization := getBiggestContainerResourceUtilization(vpa.Status.Recommendation.ContainerRecommendations)
		threshold := float32(craneAutoscaler.Spec.Behavior.VPACapacityThresholdPercent) / float32(100)
		vpaOverThreshold := biggestUtilization > threshold
		if currentlyActiveAutoscaler == refVPA {
			// If the current scaling mode is VPA we need to check if the target has reached the utilization threshold.
			// If yes, then we will switch to HPA.
			if vpaOverThreshold {
				activeAutoscaler = refHPA
				passiveAutoscaler = refVPA
				logger.Info("VPA target capacity threshold reached. Switching to HPA scaling.", "threshold", threshold, "container", containerName)
			} else {
				activeAutoscaler = refVPA
				passiveAutoscaler = refHPA
			}
		} else {
			// If the current scaling mode is HPA we need to check two things:
			//   1. Is the HPA at minimum replicas?
			//   2. Is the VPA recommendation below threshold?
			// If the answer is "yes" for both we will switch to VPA.
			hpaDesiredReplicas := hpa.Status.DesiredReplicas
			hpaMinReplicas := *hpa.Spec.MinReplicas
			hpaAtMinReplicas := hpaDesiredReplicas <= hpaMinReplicas

			if hpaAtMinReplicas && !vpaOverThreshold {
				activeAutoscaler = refVPA
				passiveAutoscaler = refHPA
				logger.Info("HPA replicas at minimum and VPA is willing to scale down. Switching to VPA scaling.", "hpaMinReplicas", hpaMinReplicas)
			} else {
				activeAutoscaler = refHPA
				passiveAutoscaler = refVPA
			}
		}
	}
	meta.SetStatusCondition(&craneAutoscaler.Status.Conditions, metav1.Condition{Type: typeScalingDecisionCraneAutoscaler, Status: metav1.ConditionTrue, Reason: activeAutoscaler, Message: fmt.Sprintf("Selected autoscaler is now %s", activeAutoscaler)})

	logger.Info("Decided which autoscaler to activate", "active", activeAutoscaler, "passive", passiveAutoscaler)

	// Reconcile VPA resource
	if err := r.reconcileVPA(ctx, craneAutoscaler, vpa, activeAutoscaler == refVPA); err != nil {
		logger.Error(err, "Failed to reconcile VPA")
		return ctrl.Result{}, err
	}

	// Reconcile HPA resource
	if err := r.reconcileHPA(ctx, craneAutoscaler, hpa, activeAutoscaler == refHPA); err != nil {
		logger.Error(err, "Failed to reconcile HPA")
		return ctrl.Result{}, err
	}

	meta.SetStatusCondition(&craneAutoscaler.Status.Conditions, metav1.Condition{Type: typeAvailableCraneAutoscaler, Status: metav1.ConditionTrue, Reason: "Reconciling", Message: "Reconciliation successful"})
	if err := r.Status().Update(ctx, craneAutoscaler); err != nil {
		logger.Error(err, "Failed to update cranepodautoscaler status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CranePodAutoscalerReconciler) getOrCreateVPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (bool, *vpav1.VerticalPodAutoscaler, error) {
	logger := log.FromContext(ctx)
	resourceKind := refVPA
	vpa := &vpav1.VerticalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: craneAutoscaler.Name, Namespace: craneAutoscaler.Namespace}, vpa)
	if err != nil && apierrors.IsNotFound(err) {
		vpa, err = r.NewVPAForAutoscaler(craneAutoscaler)
		if err != nil {
			return false, nil, r.handleAutoscalerDefinitionError(ctx, err, resourceKind, craneAutoscaler)
		}

		logger.Info("Creating a new resource",
			"resource.Kind", resourceKind, "resource.Namespace", vpa.Namespace, "resource.Name", vpa.Name)
		if err = r.Create(ctx, vpa); err != nil {
			return false, nil, r.handleAutoscalerCreationError(ctx, err, resourceKind, vpa.Namespace, vpa.Name)
		}
		return true, vpa, nil
	} else if err != nil {
		return false, nil, err
	}
	return false, vpa, nil
}

func (r *CranePodAutoscalerReconciler) getOrCreateHPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (bool, *hpav2.HorizontalPodAutoscaler, error) {
	logger := log.FromContext(ctx)
	resourceKind := refHPA
	hpa := &hpav2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: craneAutoscaler.Name, Namespace: craneAutoscaler.Namespace}, hpa)
	if err != nil && apierrors.IsNotFound(err) {
		hpa, err = r.NewHPAForAutoscaler(craneAutoscaler)
		if err != nil {
			return false, nil, r.handleAutoscalerDefinitionError(ctx, err, resourceKind, craneAutoscaler)
		}

		logger.Info("Creating a new resource",
			"resource.Kind", resourceKind, "resource.Namespace", hpa.Namespace, "resource.Name", hpa.Name)
		if err = r.Create(ctx, hpa); err != nil {
			return false, nil, r.handleAutoscalerCreationError(ctx, err, resourceKind, hpa.Namespace, hpa.Name)
		}
		return true, hpa, nil
	} else if err != nil {
		return false, nil, err
	}
	return false, hpa, nil
}

func (r *CranePodAutoscalerReconciler) handleAutoscalerCreationError(ctx context.Context, err error, resourceKind string, namespace string, name string) error {
	logger := log.FromContext(ctx)
	logger.Error(err, "Failed to create new resource", "resource.Kind", resourceKind,
		"resource.Namespace", namespace, "resource.Name", name)
	return err
}

func (r *CranePodAutoscalerReconciler) handleAutoscalerDefinitionError(ctx context.Context, err error, resourceKind string, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) error {
	logger := log.FromContext(ctx)
	logger.Error(err, "Failed to define new resource for cranepodautoscaler", "resource.Kind", resourceKind)

	// The following implementation will update the status
	meta.SetStatusCondition(&craneAutoscaler.Status.Conditions, metav1.Condition{Type: typeAvailableCraneAutoscaler,
		Status: metav1.ConditionFalse, Reason: "Reconciling",
		Message: fmt.Sprintf("Failed to create %s for the custom resource (%s): (%s)", resourceKind, craneAutoscaler.Name, err)})

	if err := r.Status().Update(ctx, craneAutoscaler); err != nil {
		logger.Error(err, "Failed to update cranepodautoscaler status")
		return err
	}

	return err
}

func (r *CranePodAutoscalerReconciler) NewVPAForAutoscaler(craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	vpa := craneAutoscaler.GenerateDisabledVPA()
	if err := ctrl.SetControllerReference(craneAutoscaler, vpa, r.Scheme); err != nil {
		return nil, err
	}
	return vpa, nil
}

func (r *CranePodAutoscalerReconciler) NewHPAForAutoscaler(craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*hpav2.HorizontalPodAutoscaler, error) {
	hpa := craneAutoscaler.GenerateEnabledHPA()
	if err := ctrl.SetControllerReference(craneAutoscaler, hpa, r.Scheme); err != nil {
		return nil, err
	}
	return hpa, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CranePodAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalingv1alpha1.CranePodAutoscaler{}).
		Owns(&hpav2.HorizontalPodAutoscaler{}).
		Owns(&vpav1.VerticalPodAutoscaler{}).
		Complete(r)
}

func getBiggestContainerResourceUtilization(vpaContainerResources []vpav1.RecommendedContainerResources) (string, float32) {
	utilization := float32(0.0)
	containerName := "NOCONTAINER"
	for _, containerResource := range vpaContainerResources {
		targetCpu := containerResource.Target.Cpu().MilliValue()
		upperBoundCpu := containerResource.UpperBound.Cpu().MilliValue()
		cpuUtilization := float32(targetCpu) / float32(upperBoundCpu)
		targetMem := containerResource.Target.Memory().Value()
		upperBoundMem := containerResource.UpperBound.Memory().Value()
		memUtilization := float32(targetMem) / float32(upperBoundMem)

		if cpuUtilization > utilization {
			utilization = cpuUtilization
			containerName = containerResource.ContainerName
		}
		if memUtilization > utilization {
			utilization = memUtilization
			containerName = containerResource.ContainerName
		}
	}

	return containerName, utilization
}

func (r *CranePodAutoscalerReconciler) reconcileVPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler, vpa *vpav1.VerticalPodAutoscaler, active bool) error {
	logger := log.FromContext(ctx)
	desiredVPA := &vpav1.VerticalPodAutoscaler{}
	if active {
		desiredVPA = craneAutoscaler.GenerateEnabledVPA()
	} else {
		desiredVPA = craneAutoscaler.GenerateDisabledVPA()
	}
	if err := ctrl.SetControllerReference(craneAutoscaler, desiredVPA, r.Scheme); err != nil {
		return err
	}
	desiredVPA.Status = vpa.Status
	if !cmp.Equal(desiredVPA.Spec, vpa.Spec) {
		logger.Info("Updating VPA", "diff", cmp.Diff(desiredVPA.Spec, vpa.Spec))
		vpa.Spec = desiredVPA.Spec
		if err := r.Update(ctx, vpa); err != nil {
			logger.Error(err, "Failed to update resource", "resource.Kind", refVPA,
				"resource.Namespace", vpa.Namespace, "resource.Name", vpa.Name)
			return err
		}
	}

	return nil
}

func (r *CranePodAutoscalerReconciler) reconcileHPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler, hpa *hpav2.HorizontalPodAutoscaler, active bool) error {
	logger := log.FromContext(ctx)
	desiredHPA := &hpav2.HorizontalPodAutoscaler{}
	if active {
		desiredHPA = craneAutoscaler.GenerateEnabledHPA()
	} else {
		desiredHPA = craneAutoscaler.GenerateDisabledHPA()
	}
	if err := ctrl.SetControllerReference(craneAutoscaler, desiredHPA, r.Scheme); err != nil {
		return err
	}
	desiredHPA.Status = hpa.Status

	if !cmp.Equal(desiredHPA.Spec, hpa.Spec) {
		logger.Info("Updating HPA", "diff", cmp.Diff(desiredHPA.Spec, hpa.Spec))
		hpa.Spec = desiredHPA.Spec
		if err := r.Update(ctx, hpa); err != nil {
			logger.Error(err, "Failed to update resource", "resource.Kind", refHPA,
				"resource.Namespace", hpa.Namespace, "resource.Name", hpa.Name)
			return err
		}
	}

	return nil
}
