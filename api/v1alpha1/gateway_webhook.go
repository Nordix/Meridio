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
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var gatewaylog = logf.Log.WithName("gateway-resource")

func (r *Gateway) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-gateway,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=gateways,verbs=create;update,versions=v1alpha1,name=mgateway.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Gateway{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Gateway) Default() {
	gatewaylog.Info("default", "name", r.Name)

	if r.Spec.Protocol == "" {
		r.Spec.Protocol = string(BGP)
	} else {
		r.Spec.Protocol = strings.ToLower(r.Spec.Protocol)
	}

	if r.Spec.Protocol == string(BGP) {
		if r.Spec.Bgp.BFD == nil {
			r.Spec.Bgp.BFD = new(bool)
		}
		if r.Spec.Bgp.HoldTime == "" {
			r.Spec.Bgp.HoldTime = "240s"
		}
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

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-gateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=gateways,verbs=create;update,versions=v1alpha1,name=vgateway.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Gateway{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Gateway) ValidateCreate() error {
	gatewaylog.Info("validate create", "name", r.Name)

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

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Gateway) validateLabels() field.ErrorList {
	var allErrs field.ErrorList
	if value, ok := r.ObjectMeta.Labels["attractor"]; !ok {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels").Child("attractor"), value, "gateway must have a attractor label"))
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
}

func (r *Gateway) validateSpec() field.ErrorList {
	var allErrs field.ErrorList

	proto := Protocol(r.Spec.Protocol)

	if !proto.IsValid() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("protocol"), r.Spec.Protocol, "protocols other than bgp is not supported yet"))
	}
	ip := net.ParseIP(r.Spec.Address)
	if ip == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("address"), r.Spec.Address, "invalid IP format"))
	}
	// if protocol is BGP
	switch proto {
	case BGP:
		// remote-asn cannot be nil
		if r.Spec.Bgp.RemoteASN == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("remote-asn"), r.Spec.Bgp.RemoteASN, "mandatory parameter is missing"))
		}
		// local-asn cannot be nil
		if r.Spec.Bgp.LocalASN == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("local-asn"), r.Spec.Bgp.LocalASN, "mandatory parameter is missing"))
		}
		// hold-time must be no less than 3 seconds
		d, err := time.ParseDuration(r.Spec.Bgp.HoldTime)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("hold-time"), r.Spec.Bgp.HoldTime, "invalid time duration format, must be a decimal number with time unit s/m/h"))
		} else {
			rounded := int(d.Seconds())
			if rounded < 3 {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("bgp").Child("hold-time"), r.Spec.Bgp.HoldTime, "invalid time duration value, must >3 seconds"))
			}
		}
		emp := StaticSpec{}
		if r.Spec.Static != emp {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("static"), r.Spec.Static, "must be empty when protocol is bgp"))
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return allErrs
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
		return apierrors.NewBadRequest(fmt.Sprintf("expected a gateway got got a %T", old))
	}
	attrNew := r.ObjectMeta.Labels["attractor"]
	attrOld := old.ObjectMeta.Labels["attractor"]
	if attrNew != attrOld {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "attractor"), "update on attractor label is forbidden"))
	}

	return nil
}
