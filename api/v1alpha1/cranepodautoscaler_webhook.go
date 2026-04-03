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
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var cranepodautoscalerlog = logf.Log.WithName("cranepodautoscaler-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *CranePodAutoscaler) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &CranePodAutoscaler{}).
		WithDefaulter(&cranePodAutoscalerDefaulter{}).
		WithValidator(&cranePodAutoscalerValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-autoscaling-phihos-github-io-v1alpha1-cranepodautoscaler,mutating=true,failurePolicy=fail,sideEffects=None,groups=autoscaling.phihos.github.io,resources=cranepodautoscalers,verbs=create;update,versions=v1alpha1,name=mcranepodautoscaler.kb.io,admissionReviewVersions=v1

// cranePodAutoscalerDefaulter implements admission.Defaulter for CranePodAutoscaler.
type cranePodAutoscalerDefaulter struct{}

var _ admission.Defaulter[*CranePodAutoscaler] = &cranePodAutoscalerDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the type.
func (d *cranePodAutoscalerDefaulter) Default(_ context.Context, obj *CranePodAutoscaler) error {
	cranepodautoscalerlog.Info("default", "name", obj.Name)

	if obj.Spec.Behavior.VPACapacityThresholdPercent == 0 {
		obj.Spec.Behavior.VPACapacityThresholdPercent = 80
	}
	return nil
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-autoscaling-phihos-github-io-v1alpha1-cranepodautoscaler,mutating=false,failurePolicy=fail,sideEffects=None,groups=autoscaling.phihos.github.io,resources=cranepodautoscalers,verbs=create;update,versions=v1alpha1,name=vcranepodautoscaler.kb.io,admissionReviewVersions=v1

// cranePodAutoscalerValidator implements admission.Validator for CranePodAutoscaler.
type cranePodAutoscalerValidator struct{}

var _ admission.Validator[*CranePodAutoscaler] = &cranePodAutoscalerValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type.
func (v *cranePodAutoscalerValidator) ValidateCreate(_ context.Context, obj *CranePodAutoscaler) (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate create", "name", obj.Name)

	return nil, obj.Validate()
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type.
func (v *cranePodAutoscalerValidator) ValidateUpdate(_ context.Context, _, obj *CranePodAutoscaler) (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate update", "name", obj.Name)

	return nil, obj.Validate()
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type.
func (v *cranePodAutoscalerValidator) ValidateDelete(_ context.Context, obj *CranePodAutoscaler) (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
