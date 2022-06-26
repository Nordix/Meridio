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
	"net"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")
var gatewayClient client.Client

func (r *Gateway) SetupWebhookWithManager(mgr ctrl.Manager) error {
	gatewayClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

var _ webhook.Defaulter = &Gateway{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Gateway) Default() {
	gatewaylog.Info("default", "name", r.Name)

	if r.Spec.Protocol == "" {
		r.Spec.Protocol = string(BGP)
	} else {
		r.Spec.Protocol = strings.ToLower(r.Spec.Protocol)
	}

	// default for BGP
	if r.Spec.Protocol == string(BGP) {
		r.Spec.Bgp.BFD = defaultBfd(BGP, r.Spec.Bgp.BFD)

		if r.Spec.Bgp.HoldTime == "" {
			r.Spec.Bgp.HoldTime = "240s"
		}
		if r.Spec.Bgp.RemotePort == nil {
			r.Spec.Bgp.RemotePort = new(uint16)
			*r.Spec.Bgp.RemotePort = 179
		}
		if r.Spec.Bgp.LocalPort == nil {
			r.Spec.Bgp.LocalPort = new(uint16)
			*r.Spec.Bgp.LocalPort = 179
		}
	}

	// default for static
	if r.Spec.Protocol == string(Static) {
		r.Spec.Static.BFD = defaultBfd(Static, r.Spec.Static.BFD)
	}
}

func defaultBfd(proto Protocol, bfd BfdSpec) BfdSpec {
	switch proto {
	case BGP:
		{
			if bfd.Switch == nil { // if bfd is empty, default to false
				bfd.Switch = new(bool)
			}
			if *bfd.Switch { // if bfd is true, fill missing BFD parameters to default value
				if bfd.MinRx == "" {
					bfd.MinRx = "300ms"
				}
				if bfd.MinTx == "" {
					bfd.MinTx = "300ms"
				}
				if bfd.Multiplier == nil {
					bfd.Multiplier = new(uint16)
					*bfd.Multiplier = 3
				}
			}
		}
	case Static:
		{
			if bfd.Switch == nil { // if bfd is empty, default to true
				bfd.Switch = new(bool)
				*bfd.Switch = true
			}
			if *bfd.Switch { // if bfd is true, fill missing BFD parameters to default value
				if bfd.MinRx == "" {
					bfd.MinRx = "200ms"
				}
				if bfd.MinTx == "" {
					bfd.MinTx = "200ms"
				}
				if bfd.Multiplier == nil {
					bfd.Multiplier = new(uint16)
					*bfd.Multiplier = 3
				}
			}
		}
	}
	return bfd
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=gateways,verbs=create;update,versions=v1alpha1,name=vgateway.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Gateway{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateCreate() error {
	gatewaylog.Info("validate create", "name", r.Name)
	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := gatewayClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}
	return r.validateGateway()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateUpdate(old runtime.Object) error {
	gatewaylog.Info("validate update", "name", r.Name)

	err := r.validateUpdate(old)
	if err != nil {
		return err
	}
	return r.validateGateway()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateDelete() error {
	gatewaylog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Gateway) validateLabels() field.ErrorList {
	var allErrs field.ErrorList
	if value, ok := r.ObjectMeta.Labels["trench"]; !ok {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels").Child("trench"), value, fmt.Sprintf("%s must have a trench label", r.GroupVersionKind().Kind)))
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func (r *Gateway) validateSpec() field.ErrorList {
	var allErrs field.ErrorList

	proto := Protocol(r.Spec.Protocol)

	ip := net.ParseIP(r.Spec.Address)
	if ip == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("address"), r.Spec.Address, "invalid IP format"))
	}
	bfdEmty := BfdSpec{}
	// if protocol is BGP
	switch proto {
	case BGP:
		if r.Spec.Bgp.RemoteASN == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("remote-asn"), r.Spec.Bgp.RemoteASN, "mandatory parameter is missing"))
		}
		if r.Spec.Bgp.LocalASN == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("local-asn"), r.Spec.Bgp.LocalASN, "mandatory parameter is missing"))
		}
		if r.Spec.Bgp.RemotePort == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("remote-port"), r.Spec.Bgp.RemotePort, "mandatory parameter is missing"))
		}
		if r.Spec.Bgp.LocalPort == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("local-port"), r.Spec.Bgp.LocalPort, "mandatory parameter is missing"))
		}
		if r.Spec.Bgp.BFD != bfdEmty {
			allErrs = append(allErrs, bfdValidation("bgp", r.Spec.Bgp.BFD)...)
		}
		// hold-time must be no less than 3 seconds
		if err := timeInBound(r.Spec.Bgp.HoldTime, 3*time.Second); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("hold-time"), r.Spec.Bgp.HoldTime, err.Error()))
		}
		emp := StaticSpec{}
		if r.Spec.Static != emp {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("static"), r.Spec.Static, "must be empty when protocol is bgp"))
		}
	// if protocol is static
	case Static:
		emp := BgpSpec{}
		if r.Spec.Bgp != emp {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp"), r.Spec.Bgp, "must be empty when protocol is static"))
		}
		if r.Spec.Static.BFD != bfdEmty {
			allErrs = append(allErrs, bfdValidation("static", r.Spec.Static.BFD)...)
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func bfdValidation(proto string, bfd BfdSpec) field.ErrorList {
	var allErrs field.ErrorList
	if bfd.Switch == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("switch"), nil, "switch must be specified"))
		return allErrs
	}
	if !*bfd.Switch {
		return allErrs
	}
	if bfd.MinRx == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("min-rx"), bfd.MinRx, "min-rx must be specified"))
	}
	if bfd.MinTx == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("min-tx"), bfd.MinTx, "min-tx must be specified"))
	}
	if bfd.Multiplier == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("multiplier"), bfd.Multiplier, "multiplier must be specified"))
	}

	if err := timeInBound(bfd.MinRx, 0*time.Second); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("min-rx"), bfd.MinRx, err.Error()))
	}
	if err := timeInBound(bfd.MinTx, 0*time.Second); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child(proto).Child("bfd").Child("min-tx"), bfd.MinTx, err.Error()))
	}
	return allErrs
}

// validate if the timer is above the lower bound
func timeInBound(t string, bound time.Duration) error {
	d, err := time.ParseDuration(t)
	if err != nil {
		return fmt.Errorf("invalid time duration format, must be a decimal number with time unit ms/s/m/h")
	} else {
		rounded := int64(d.Milliseconds())
		if rounded < bound.Milliseconds() {
			return fmt.Errorf("invalid time duration value, must > %s", bound.String())
		}
	}
	return nil
}

func (r *Gateway) validateGateway() error {
	var allErrs field.ErrorList
	if err := r.validateLabels(); err != nil {
		allErrs = append(allErrs, err...)
	}
	if err := r.validateSpec(); err != nil {
		allErrs = append(allErrs, err...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupKind(), r.Name, allErrs)
}

func (r *Gateway) validateUpdate(oldObj runtime.Object) error {
	old, ok := oldObj.(*Gateway)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a %s got a %T", r.GroupVersionKind().Kind, old))
	}
	attrNew := r.ObjectMeta.Labels["trench"]
	attrOld := old.ObjectMeta.Labels["trench"]
	if attrNew != attrOld {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on trench label is forbidden"))
	}

	return nil
}
