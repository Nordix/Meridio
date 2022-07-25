/*
Copyright (c) 2021-2022 Nordix Foundation

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
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var viplog = logf.Log.WithName("vip-resource")
var vipClient client.Client

func (r *Vip) SetupWebhookWithManager(mgr ctrl.Manager) error {
	vipClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-vip,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=vips,verbs=create;update,versions=v1alpha1,name=vvip.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Vip{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Vip) ValidateCreate() error {
	viplog.Info("validate create", "name", r.Name)

	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := vipClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}
	return r.validateVip()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Vip) ValidateUpdate(old runtime.Object) error {
	viplog.Info("validate update", "name", r.Name)
	err := r.validateLabelUpdate(old)
	if err != nil {
		return err
	}
	return r.validateVip()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Vip) ValidateDelete() error {
	viplog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Vip) validateVip() error {
	var allErrs field.ErrorList
	if _, err := validatePrefix(r.Spec.Address); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("address"), r.Spec.Address, err.Error()))
	}

	if err := r.validateLabels(); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		r.GroupKind(), r.Name, allErrs)
}

func (r *Vip) validateLabels() error {
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		return fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
	}
	return nil
}

func (r *Vip) validateLabelUpdate(oldObj runtime.Object) error {
	vipOld, ok := oldObj.(*Vip)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a %s got a %T", r.GroupVersionKind().Kind, vipOld))
	}
	new := r.ObjectMeta.Labels["trench"]
	old := vipOld.ObjectMeta.Labels["trench"]
	if new != old {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on vip label trench is forbidden"))
	}
	return nil
}
