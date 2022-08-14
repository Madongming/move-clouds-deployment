/*
Copyright 2022.

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

package v1

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	ServiceNodePort = "nodeport"
	ServiceIngress  = "ingress"
)

// log is for logging in this package.
var singledeploymentlog = logf.Log.WithName("singledeployment-resource")

func (r *SingleDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-deployment-github-com-v1-singledeployment,mutating=true,failurePolicy=fail,sideEffects=None,groups=deployment.github.com,resources=singledeployments,verbs=create;update,versions=v1,name=msingledeployment.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &SingleDeployment{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *SingleDeployment) Default() {
	singledeploymentlog.Info("default", "name", r.Name)

	if r.Spec.Replicas == 0 {
		r.Spec.Replicas = 1
	}

	if r.Spec.Expose.ServicePort == 0 {
		r.Spec.Expose.ServicePort = r.Spec.Port
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-deployment-github-com-v1-singledeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=deployment.github.com,resources=singledeployments,verbs=create;update,versions=v1,name=vsingledeployment.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &SingleDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SingleDeployment) ValidateCreate() error {
	singledeploymentlog.Info("validate create", "name", r.Name)

	return r.validateCreateAndUpdate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SingleDeployment) ValidateUpdate(_ runtime.Object) error {
	singledeploymentlog.Info("validate update", "name", r.Name)

	return r.validateCreateAndUpdate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SingleDeployment) ValidateDelete() error {
	singledeploymentlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *SingleDeployment) validateCreateAndUpdate() error {
	errs := field.ErrorList{}
	exposePath := field.NewPath("spec", "expose")
	if strings.ToLower(r.Spec.Expose.Mode) != ServiceNodePort &&
		strings.ToLower(r.Spec.Expose.Mode) != ServiceIngress {
		errs = append(errs,
			field.NotSupported(exposePath.Child("mode"), r.Spec.Expose.Mode, []string{ServiceIngress, ServiceNodePort}))
	}

	if strings.ToLower(r.Spec.Expose.Mode) == ServiceNodePort &&
		(r.Spec.Expose.NodePort == 0 ||
			r.Spec.Expose.NodePort > 32767 ||
			r.Spec.Expose.NodePort < 30000) {
		errs = append(errs,
			field.Invalid(exposePath.Child("nodePort"), r.Spec.Expose.NodePort, "If spec.expose.mode is `NodePort`, the `spec.expose.nodePort` must not be empty, and it must be in 30000-32767"))
	}

	if strings.ToLower(r.Spec.Expose.Mode) == ServiceIngress &&
		r.Spec.Expose.IngressDomain == "" {
		errs = append(errs,
			field.Invalid(exposePath.Child("ingressDomain"), r.Spec.Expose.NodePort, "If spec.expose.mode is `ingress`, the `spec.expose.ingressDomain` must not be empty "))
	}

	if len(errs) != 0 {
		return errs.ToAggregate()
	}

	return nil
}
