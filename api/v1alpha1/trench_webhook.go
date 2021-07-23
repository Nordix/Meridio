/*
Copyright 2021.

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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var trenchlog = logf.Log.WithName("trench-resource")

func (r *Trench) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-trench,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=trenches,verbs=create;update,versions=v1alpha1,name=mtrench.kb.io,admissionReviewVersions={v1alpha1,v1beta1}

var _ webhook.Defaulter = &Trench{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Trench) Default() {
	trenchlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-trench,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=trenches,verbs=create;update,versions=v1alpha1,name=vtrench.kb.io,admissionReviewVersions={v1alpha1,v1beta1}

var _ webhook.Validator = &Trench{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Trench) ValidateCreate() error {
	trenchlog.Info("validate create", "name", r.Name)
	return r.validateTrench()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Trench) ValidateUpdate(old runtime.Object) error {
	trenchlog.Info("validate update", "name", r.Name)
	return r.validateTrench()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Trench) ValidateDelete() error {
	trenchlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Trench) validateTrench() error {
	var allErrs field.ErrorList

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "meridio.nordix.org", Kind: "Trench"},
		r.Name, allErrs)
}
