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
	"context"
	"fmt"
	"net"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var attractorlog = logf.Log.WithName("attractor-resource")

func (r *Attractor) SetupWebhookWithManager(mgr ctrl.Manager) error {
	attractorClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-attractor,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=attractors,verbs=create;update,versions=v1alpha1,name=mattractor.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Attractor{}
var attractorClient client.Client

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Attractor) Default() {
	attractorlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-attractor,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=attractors,verbs=create;update,versions=v1alpha1,name=vattractor.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Attractor{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Attractor) ValidateCreate() error {
	attractorlog.Info("validate create", "name", r.Name)

	// Get the trench by the label in attractor
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := attractorClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}

	// validation: get all attractors with same trench, verdict the number should not be greater than 1
	al := &AttractorList{}
	sel := labels.Set{"trench": trench.ObjectMeta.Name}
	err = attractorClient.List(context.TODO(), al, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})

	if err != nil {
		return fmt.Errorf("unable to get attractors")
	} else if len(al.Items) >= 1 {
		var names []string
		for _, a := range al.Items {
			names = append(names, a.ObjectMeta.Name)
		}
		return fmt.Errorf("only one attractor is allowed in a trench, but also found %s", strings.Join(names, ", "))
	}

	return r.validateAttractor()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Attractor) ValidateUpdate(old runtime.Object) error {
	attractorlog.Info("validate update", "name", r.Name)
	err := r.validateUpdate(old)
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

	_, n, err := net.ParseCIDR(r.Spec.VlanPrefixIPv4)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vlan-ipv4-prefix"), r.Spec.VlanPrefixIPv4, err.Error()))
	}
	if n.String() != r.Spec.VlanPrefixIPv4 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vlan-ipv4-prefix"), r.Spec.VlanPrefixIPv4, fmt.Sprintf("not a valid prefix, probably %v should be used", n)))
	}

	_, n, err = net.ParseCIDR(r.Spec.VlanPrefixIPv6)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vlan-ipv6-prefix"), r.Spec.VlanPrefixIPv6, err.Error()))
	}
	if n.String() != r.Spec.VlanPrefixIPv6 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vlan-ipv6-prefix"), r.Spec.VlanPrefixIPv6, fmt.Sprintf("not a valid prefix, probably %v should be used", n)))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: r.GroupVersionKind().Group, Kind: r.GroupVersionKind().Kind},
		r.Name, allErrs)
}

func (r *Attractor) validateLabels() error {
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		return fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
	}
	return nil
}

func (r *Attractor) validateUpdate(old runtime.Object) error {
	attrOld, ok := old.(*Attractor)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a %s got a %T", r.GroupVersionKind().Kind, attrOld))
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
			r.Name, field.Forbidden(field.NewPath("spec", "vlan-interface"), "update on vlan interface is forbidden"))
	}

	if r.Spec.VlanPrefixIPv4 != attrOld.Spec.VlanPrefixIPv4 {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "vlan-ipv4-prefix"), "update on vlan prefix is forbidden"))
	}

	if r.Spec.VlanPrefixIPv6 != attrOld.Spec.VlanPrefixIPv6 {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "vlan-ipv6-prefix"), "update on vlan prefix is forbidden"))
	}
	return nil
}
