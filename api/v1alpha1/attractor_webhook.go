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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var attractorlog = logf.Log.WithName("attractor-resource")

func (r *Attractor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-attractor,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=attractors,verbs=create;update,versions=v1alpha1,name=mattractor.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Attractor{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Attractor) Default() {
	attractorlog.Info("default", "name", r.Name)

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = new(int32)
		*r.Spec.Replicas = 1
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-attractor,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=attractors,verbs=create;update,versions=v1alpha1,name=vattractor.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Attractor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Attractor) ValidateCreate() error {
	attractorlog.Info("validate create", "name", r.Name)
	return r.validateAttractor()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Attractor) ValidateUpdate(old runtime.Object) error {
	attractorlog.Info("validate update", "name", r.Name)
	err := r.validateLabelUpdate(old)
	if err != nil {
		return err
	}
	return r.validateAttractor()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Attractor) ValidateDelete() error {
	attractorlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Attractor) validateAttractor() error {
	var allErrs field.ErrorList
	if err := r.validateLabels(); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "meridio.nordix.org", Kind: "Attactor"},
		r.Name, allErrs)
}

func (r *Attractor) validateLabels() error {
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		return fmt.Errorf("attactor must have a trench label")
	}
	return nil
}

func (r *Attractor) validateLabelUpdate(old runtime.Object) error {
	attrOld, ok := old.(*Attractor)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a attractor got got a %T", attrOld))
	}
	trenchNew := r.ObjectMeta.Labels["trench"]
	trenchOld := attrOld.ObjectMeta.Labels["trench"]
	if trenchNew != trenchOld {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on attractor label trench is forbidden"))
	}

	if r.Spec.VlanID != attrOld.Spec.VlanID {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "vlan-id"), "update on vlan id is forbidden"))
	}

	if r.Spec.VlanInterface != attrOld.Spec.VlanInterface {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "vlan-interface"), "update on vlan interface is forbidden"))
	}
	return nil
}
