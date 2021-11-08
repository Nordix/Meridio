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
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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

//+kubebuilder:webhook:path=/mutate-meridio-nordix-org-v1alpha1-conduit,mutating=true,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=conduits,verbs=create;update,versions=v1alpha1,name=mconduit.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Conduit{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Conduit) Default() {
	conduitlog.Info("default", "name", r.Name)

	if r.Spec.Type == "" {
		r.Spec.Type = string(StatelessLB)
	} else {
		r.Spec.Type = strings.ToLower(r.Spec.Type)
	}

	if r.Spec.Replicas == nil {
		r.Spec.Replicas = new(int32)
		*r.Spec.Replicas = 1
	}
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

	// workaround for lb and fe are in the same deployment, the env vars come from both conduit and attractor
	al := &AttractorList{}
	sel := labels.Set{"trench": trench.ObjectMeta.Name}
	err = conduitClient.List(context.TODO(), al, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})
	if err != nil || len(al.Items) != 1 {
		return fmt.Errorf("conduit must be created when there is one and only one attractor in the same trench")
	}

	// validation: get all attractors with same trench, verdict the number should not be greater than 1
	cl := &ConduitList{}
	sel = labels.Set{"trench": trench.ObjectMeta.Name}
	err = attractorClient.List(context.TODO(), cl, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})

	if err != nil {
		return fmt.Errorf("unable to get %s", r.GroupKind().Kind)
	} else if len(cl.Items) >= 1 {
		var names []string
		for _, a := range cl.Items {
			names = append(names, a.ObjectMeta.Name)
		}
		return fmt.Errorf("only one conduit is allowed in a trench, but also found %s", strings.Join(names, ", "))
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
	if !NetworkServiceType(r.Spec.Type).IsValid() {
		err := fmt.Errorf("invalid value")
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("spec").Child("lb-network-service"), r.Spec.Type, err.Error()))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupKind(), r.Name, allErrs)
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
