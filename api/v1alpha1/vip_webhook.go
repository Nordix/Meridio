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
	"net"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var viplog = logf.Log.WithName("vip-resource")

func (r *Vip) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-vip,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=vips,verbs=create;update,versions=v1alpha1,name=vvip.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Vip{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Vip) ValidateCreate() error {
	viplog.Info("validate create", "name", r.Name)

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
	if err := r.validateAddresses(); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("address"), r.Spec.Address, err.Error()))
	}

	if err := r.validateLabels(); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "meridio.nordix.org", Kind: "Vip"},
		r.Name, allErrs)
}

func (r *Vip) validateAddresses() error {
	_, _, err := net.ParseCIDR(r.Spec.Address)
	if err != nil {
		return err
	}
	ip, ipnet, err := net.ParseCIDR(r.Spec.Address)
	if err != nil {
		return err
	}
	// ipv4 cidr validation for alpha
	if ip.To4() != nil && ipnet.Mask.String() != net.CIDRMask(32, 32).String() {
		return fmt.Errorf("only /32 address is supported for ipv4 vips")
	}
	// ipv6 cidr validation for alpha
	if ip.To4() == nil && ipnet.Mask.String() != net.CIDRMask(128, 128).String() {
		return fmt.Errorf("only /128 address is supported for ipv6 vips")
	}
	return nil
}

func (r *Vip) validateLabels() error {
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		return fmt.Errorf("vip must have a trench label")
	}
	return nil
}

func (r *Vip) validateLabelUpdate(oldObj runtime.Object) error {
	vipOld, ok := oldObj.(*Vip)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a vip got got a %T", vipOld))
	}
	new := r.ObjectMeta.Labels["trench"]
	old := vipOld.ObjectMeta.Labels["trench"]
	if new != old {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on vip label trench is forbidden"))
	}
	new = r.ObjectMeta.Labels["attractor"]
	old = vipOld.ObjectMeta.Labels["attractor"]
	if new != old {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "attractor"), "update on vip label attractor is forbidden"))
	}
	return nil
}
