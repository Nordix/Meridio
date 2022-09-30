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
var conduitlog = logf.Log.WithName("conduit-resource")
var conduitClient client.Client

func (r *Conduit) SetupWebhookWithManager(mgr ctrl.Manager) error {
	conduitClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-conduit,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=conduits,verbs=create;update,versions=v1alpha1,name=vconduit.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Conduit{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Conduit) ValidateCreate() error {
	conduitlog.Info("validate create", "name", r.Name)
	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := conduitClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}

	return r.validateConduit()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Conduit) ValidateUpdate(old runtime.Object) error {
	conduitlog.Info("validate update", "name", r.Name)

	err := r.validateUpdate(old)
	if err != nil {
		return err
	}
	return r.validateConduit()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Conduit) ValidateDelete() error {
	conduitlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Conduit) validateConduit() error {
	var allErrs field.ErrorList
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		err := fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}
	if r.Spec.Type == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("type"), r.Spec.Type, "cannot be empty"))
	}
	err := r.validatePortNat()
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("destination-port-nats"), r.Spec.DestinationPortNats,
			fmt.Sprintf("destination port nats: %s", err.Error())))
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupKind(), r.Name, allErrs)
}

func (r *Conduit) validatePortNat() error {
	targetPortVipProtocolMap := map[string]int{}
	portVipProtocolMap := map[string]int{}
	portTargetPortProtocolMap := map[string]int{}
	for i, pn := range r.Spec.DestinationPortNats {
		portTargetPortProtocol := fmt.Sprintf("%d-%d-%s", pn.Port, pn.TargetPort, pn.Protocol)
		index, exists := portTargetPortProtocolMap[portTargetPortProtocol]
		if exists {
			return fmt.Errorf("[%d] and [%d] must be merged", index, i)
		}
		portTargetPortProtocolMap[portTargetPortProtocol] = i
		for _, vip := range pn.Vips {
			targetPortVipProtocol := fmt.Sprintf("%d-%s-%s", pn.TargetPort, vip, pn.Protocol)
			portVipProtocol := fmt.Sprintf("%d-%s-%s", pn.Port, vip, pn.Protocol)
			index, exists := targetPortVipProtocolMap[targetPortVipProtocol]
			if exists {
				return portNatCollisionError(r.Spec.DestinationPortNats, index, i)
			}
			index, exists = targetPortVipProtocolMap[portVipProtocol]
			if exists {
				return portNatCollisionError(r.Spec.DestinationPortNats, index, i)
			}
			index, exists = portVipProtocolMap[targetPortVipProtocol]
			if exists {
				return portNatCollisionError(r.Spec.DestinationPortNats, index, i)
			}
			index, exists = portVipProtocolMap[portVipProtocol]
			if exists {
				return portNatCollisionError(r.Spec.DestinationPortNats, index, i)
			}
			targetPortVipProtocolMap[targetPortVipProtocol] = i
			portVipProtocolMap[portVipProtocol] = i
		}
	}
	return nil
}

func portNatCollisionError(destinationPortNats []PortNatSpec, indexA int, indexB int) error {
	return fmt.Errorf("[%d] and [%d] are colliding", indexA, indexB)
}

func (r *Conduit) validateUpdate(oldObj runtime.Object) error {
	old, ok := oldObj.(*Conduit)
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
