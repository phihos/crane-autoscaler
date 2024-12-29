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
	// typeAvailableCraneAutoscaler represents the status of the Deployment reconciliation
	typeAvailableCraneAutoscaler = "Available"
	// typeDegradedCraneAutoscaler represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	// typeDegradedCraneAutoscaler = "Degraded"
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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CranePodAutoscaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
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
		if err = r.Status().Update(ctx, craneAutoscaler); err != nil {
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

	// Get or create VPA.
	vpa, err := r.getOrCreateVPA(ctx, craneAutoscaler)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Get or create HPA.
	hpa, err := r.getOrCreateHPA(ctx, craneAutoscaler)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Got VPA and HPA", "VPA", vpa.Name, "HPA", hpa.Name)

	return ctrl.Result{}, nil
}

func (r *CranePodAutoscalerReconciler) getOrCreateVPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	logger := log.FromContext(ctx)
	resourceKind := "VPA"
	vpa := &vpav1.VerticalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: craneAutoscaler.Name, Namespace: craneAutoscaler.Namespace}, vpa)
	if err != nil && apierrors.IsNotFound(err) {
		vpa, err = r.VPAForAutoscaler(craneAutoscaler)
		if err != nil {
			return nil, r.handleAutoscalerDefinitionError(ctx, err, resourceKind, craneAutoscaler)
		}

		logger.Info("Creating a new resource",
			"resource.Kind", resourceKind, "resource.Namespace", vpa.Namespace, "resource.Name", vpa.Name)
		if err = r.Create(ctx, vpa); err != nil {
			return nil, r.handleAutoscalerCreationError(ctx, err, resourceKind, vpa.Namespace, vpa.Name)
		}
	} else if err != nil {
		return nil, err
	}
	return vpa, nil
}

func (r *CranePodAutoscalerReconciler) getOrCreateHPA(ctx context.Context, craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*hpav2.HorizontalPodAutoscaler, error) {
	logger := log.FromContext(ctx)
	resourceKind := "HPA"
	hpa := &hpav2.HorizontalPodAutoscaler{}
	err := r.Get(ctx, types.NamespacedName{Name: craneAutoscaler.Name, Namespace: craneAutoscaler.Namespace}, hpa)
	if err != nil && apierrors.IsNotFound(err) {
		hpa, err = r.HPAForAutoscaler(craneAutoscaler)
		if err != nil {
			return nil, r.handleAutoscalerDefinitionError(ctx, err, resourceKind, craneAutoscaler)
		}

		logger.Info("Creating a new resource",
			"resource.Kind", resourceKind, "resource.Namespace", hpa.Namespace, "resource.Name", hpa.Name)
		if err = r.Create(ctx, hpa); err != nil {
			return nil, r.handleAutoscalerCreationError(ctx, err, resourceKind, hpa.Namespace, hpa.Name)
		}
	} else if err != nil {
		return nil, err
	}
	return hpa, nil
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

func (r *CranePodAutoscalerReconciler) VPAForAutoscaler(craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      craneAutoscaler.Name,
			Namespace: craneAutoscaler.Namespace,
		},
		Spec: craneAutoscaler.Spec.VPA,
	}
	if err := ctrl.SetControllerReference(craneAutoscaler, vpa, r.Scheme); err != nil {
		return nil, err
	}
	return vpa, nil
}

func (r *CranePodAutoscalerReconciler) HPAForAutoscaler(craneAutoscaler *autoscalingv1alpha1.CranePodAutoscaler) (*hpav2.HorizontalPodAutoscaler, error) {
	hpa := &hpav2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      craneAutoscaler.Name,
			Namespace: craneAutoscaler.Namespace,
		},
		Spec: craneAutoscaler.Spec.HPA,
	}
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
