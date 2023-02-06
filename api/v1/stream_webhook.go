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

package v1

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
var streamlog = logf.Log.WithName("stream-resource")
var streamClient client.Client

func (r *Stream) SetupWebhookWithManager(mgr ctrl.Manager) error {
	streamClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1-stream,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=streams,verbs=create;update,versions=v1,name=vstream.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Stream{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Stream) ValidateCreate() error {
	streamlog.Info("validate create", "name", r.Name)

	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := streamClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}

	return r.validateStream()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Stream) ValidateUpdate(old runtime.Object) error {
	streamlog.Info("validate update", "name", r.Name)

	err := r.validateUpdate(old)
	if err != nil {
		return err
	}
	return r.validateStream()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Stream) ValidateDelete() error {
	streamlog.Info("validate delete", "name", r.Name)

	return nil
}

func (r *Stream) validateStream() error {
	var allErrs field.ErrorList
	var err error
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		err = fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
	}
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}
	if r.Spec.MaxTargets != nil && (*r.Spec.MaxTargets < 1 || *r.Spec.MaxTargets > 10000) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("max-targets"), r.Spec.MaxTargets, "must be greater than or equal to 1 and less than or equal to 10000"))
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(r.GroupKind(), r.Name, allErrs)
}

func (r *Stream) validateUpdate(oldObj runtime.Object) error {
	old, ok := oldObj.(*Stream)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a %s got a %T", r.GroupVersionKind().Kind, old))
	}
	attrNew := r.ObjectMeta.Labels["trench"]
	attrOld := old.ObjectMeta.Labels["trench"]
	if attrNew != attrOld {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on trench label is forbidden"))
	}
	if r.Spec.MaxTargets != old.Spec.MaxTargets &&
		(r.Spec.MaxTargets == nil || old.Spec.MaxTargets == nil || *r.Spec.MaxTargets != *old.Spec.MaxTargets) {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "spec", "max-targets"), "update on max-targets is forbidden"))
	}
	return nil
}
