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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var cranepodautoscalerlog = logf.Log.WithName("cranepodautoscaler-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *CranePodAutoscaler) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-autoscaling-phihos-github-io-v1alpha1-cranepodautoscaler,mutating=true,failurePolicy=fail,sideEffects=None,groups=autoscaling.phihos.github.io,resources=cranepodautoscalers,verbs=create;update,versions=v1alpha1,name=mcranepodautoscaler.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &CranePodAutoscaler{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *CranePodAutoscaler) Default() {
	cranepodautoscalerlog.Info("default", "name", r.Name)

}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-autoscaling-phihos-github-io-v1alpha1-cranepodautoscaler,mutating=false,failurePolicy=fail,sideEffects=None,groups=autoscaling.phihos.github.io,resources=cranepodautoscalers,verbs=create;update,versions=v1alpha1,name=vcranepodautoscaler.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CranePodAutoscaler{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CranePodAutoscaler) ValidateCreate() (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate create", "name", r.Name)

	return nil, r.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CranePodAutoscaler) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate update", "name", r.Name)

	return nil, r.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CranePodAutoscaler) ValidateDelete() (admission.Warnings, error) {
	cranepodautoscalerlog.Info("validate delete", "name", r.Name)

	return nil, nil
}
