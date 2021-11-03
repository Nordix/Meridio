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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var flowlog = logf.Log.WithName("flow-resource")
var flowClient client.Client

func (r *Flow) SetupWebhookWithManager(mgr ctrl.Manager) error {
	flowClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-flow,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=flows,verbs=create;update,versions=v1alpha1,name=mflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &Flow{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Flow) Default() {
	flowlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1alpha1-flow,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=flows,verbs=create;update,versions=v1alpha1,name=vflow.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &Flow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateCreate() error {
	flowlog.Info("validate create", "name", r.Name)

	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := flowClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}
	return r.validateFlow()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateUpdate(old runtime.Object) error {
	flowlog.Info("validate update", "name", r.Name)

	err := r.validateUpdate(old)
	if err != nil {
		return err
	}
	return r.validateFlow()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateDelete() error {
	flowlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Flow) validateFlow() error {
	var allErrs field.ErrorList
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		err := fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}

	if len(r.Spec.Protocols) > 2 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("protocols"), r.Spec.Protocols, "only TCP and UDP are supported"))
	} else if len(r.Spec.Protocols) == 2 {
		if r.Spec.Protocols[0] == r.Spec.Protocols[1] {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("protocols"), r.Spec.Protocols, "duplicated protocols"))
		}
	}

	for i, p := range r.Spec.Protocols {
		if !TransportProtocol(p).IsValid() {
			err := fmt.Errorf("protocol[%d]: %s is invalid", i, p)
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("protocols"), p, err.Error()))
		}
	}

	if n, err := validateSubnets(r.Spec.SourceSubnets); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("source-subnets"), n,
			fmt.Sprintf("source subnet%s", err.Error())))
	}

	if p, err := validatePorts(r.Spec.SourcePorts); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("source-ports"), p,
			fmt.Sprintf("source port%s", err.Error())))
	}

	if p, err := validatePorts(r.Spec.DestinationPorts); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("destination-ports"), p,
			fmt.Sprintf("destination port%s", err.Error())))
	}

	if r.Spec.Priority < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("priority"), r.Spec.Priority,
			"priority must be larger than 0"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupKind(), r.Name, allErrs)
}

func validatePorts(ports []string) (string, error) {
	var portsList []Ports
	for i, p := range ports {
		if candidatePorts, err := validPortsFormat(p); err != nil {
			return p, fmt.Errorf("[%d]: %s", i, err.Error())
		} else {
			// append candidatePorts to portsList if there's no overlapping
			// the portsList will be used to test overlapping for the next candidate port
			if portsList, err = checkPortsOverlapping(portsList, candidatePorts); err != nil {
				return p, fmt.Errorf("[%d]: %s", i, err.Error())
			}
		}
	}
	return "", nil
}

func checkPortsOverlapping(allPorts []Ports, candidatePort Ports) ([]Ports, error) {
	if len(allPorts) == 0 {
		return insertPortList(allPorts, 0, candidatePort), nil
	}
	for j, validp := range allPorts {
		if candidatePort.Start > validp.End {
			if j == len(allPorts)-1 {
				allPorts = insertPortList(allPorts, len(allPorts), candidatePort)
			}
			continue
		} else if candidatePort.End < validp.Start {
			allPorts = insertPortList(allPorts, j, candidatePort)
		} else {
			return allPorts, fmt.Errorf("overlapping ports")
		}
	}
	return allPorts, nil
}

func insertPortList(pl []Ports, i int, p Ports) []Ports {
	if i == len(pl) {
		return append(pl, p)
	} else {
		pl = append(pl[:i+1], pl[i:]...)
		pl[i] = p
		return pl
	}
}

func validateSubnets(subnets []string) (string, error) {
	var allNonOverlappingSubnets []*net.IPNet
	for i, s := range subnets {
		n, err := validatePrefix(s)
		if err != nil {
			return s, fmt.Errorf("[%d]: %s", i, err.Error())
		}

		for j, m := range allNonOverlappingSubnets {
			if subnetsOverlap(n, m) {
				return s, fmt.Errorf("[%d] and [%d]: %s", i, j, "overlapping subnet")
			}
		}
		allNonOverlappingSubnets = append(allNonOverlappingSubnets, n)
	}
	return "", nil
}

func (r *Flow) validateUpdate(oldObj runtime.Object) error {
	old, ok := oldObj.(*Flow)
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
