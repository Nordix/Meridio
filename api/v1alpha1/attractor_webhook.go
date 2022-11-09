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

var attractorClient client.Client

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-attractor,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=attractors,verbs=create;update,versions=v1alpha1,name=vattractor.kb.io,admissionReviewVersions=v1

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

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = new(int32)
		*r.Spec.Replicas = 1
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
	_, err := validatePrefix(r.Spec.Interface.PrefixIPv4)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("ipv4-prefix"), r.Spec.Interface.PrefixIPv4, err.Error()))
	}

	_, err = validatePrefix(r.Spec.Interface.PrefixIPv6)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("ipv6-prefix"), r.Spec.Interface.PrefixIPv6, err.Error()))
	}

	switch r.Spec.Interface.Type {
	case NSMVlan:
		if r.Spec.Interface.NSMVlan.BaseInterface == "" || r.Spec.Interface.NSMVlan.VlanID == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("interface").Child("nsm-vlan"),
				r.Spec.Interface.NSMVlan, "missing mandatory parameter base-interface/vlan-id"))
		}
	default:
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("interface").Child("type"), r.Spec.Interface.Type, "not a supported interface"))
	}

	if len(r.Spec.Composites) > 1 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("composites"), r.Spec.Composites, "only one composite per attractor is supported in the current version"))
	}

	al := &AttractorList{}
	sel := labels.Set{"trench": r.ObjectMeta.Labels["trench"]}
	err = attractorClient.List(context.TODO(), al, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("composites"), r.Spec.Composites, "unable to get attractors"))
	} else {
		vips := map[string]struct{}{}
		composites := map[string]struct{}{}
		for _, vip := range r.Spec.Vips {
			vips[vip] = struct{}{}
		}
		for _, composite := range r.Spec.Composites {
			composites[composite] = struct{}{}
		}
		for _, attractor := range al.Items {
			for _, vip := range attractor.Spec.Vips {
				if attractor.ObjectMeta.Name == r.ObjectMeta.Name && attractor.ObjectMeta.Namespace == r.ObjectMeta.Namespace {
					continue
				}
				_, exists := vips[vip]
				if exists {
					allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vips"), r.Spec.Vips, "a vip cannot be shared between 2 attractors in this version"))
				}
			}
			for _, composite := range attractor.Spec.Composites {
				if attractor.ObjectMeta.Name == r.ObjectMeta.Name && attractor.ObjectMeta.Namespace == r.ObjectMeta.Namespace {
					continue
				}
				_, exists := composites[composite]
				if exists {
					allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("composites"), r.Spec.Composites, "a conduit cannot be shared between 2 attractors in this version"))
				}
			}
		}
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

	if r.Spec.Interface.Type != strings.ToLower(attrOld.Spec.Interface.Type) {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "interface", "type"), "update on interface type is forbidden"))
	}

	switch r.Spec.Interface.Type {
	case NSMVlan:
		if *(r.Spec.Interface.NSMVlan.VlanID) != *(attrOld.Spec.Interface.NSMVlan.VlanID) {
			return apierrors.NewForbidden(r.GroupResource(),
				r.Name, field.Forbidden(field.NewPath("spec", "interface", "nsm-vlan", "vlan-id"), "update on vlan id is forbidden"))
		}

		if r.Spec.Interface.NSMVlan.BaseInterface != attrOld.Spec.Interface.NSMVlan.BaseInterface {
			return apierrors.NewForbidden(r.GroupResource(),
				r.Name, field.Forbidden(field.NewPath("spec", "interface", "nsm-vlan", "base-interface"), "update on base interface is forbidden"))
		}
	}

	if r.Spec.Interface.PrefixIPv4 != attrOld.Spec.Interface.PrefixIPv4 {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "ipv4-prefix"), "update on prefix is forbidden"))
	}

	if r.Spec.Interface.PrefixIPv6 != attrOld.Spec.Interface.PrefixIPv6 {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("spec", "ipv6-prefix"), "update on prefix is forbidden"))
	}
	return nil
}
