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
	"net"
	"strings"

	"github.com/nordix/meridio/pkg/configuration/reader"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

//+kubebuilder:webhook:path=/validate-meridio-nordix-org-v1-flow,mutating=false,failurePolicy=fail,sideEffects=None,groups=meridio.nordix.org,resources=flows,verbs=create;update,versions=v1,name=vflow.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Flow{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateCreate() (admission.Warnings, error) {
	flowlog.Info("validate create", "name", r.Name, "stream", r.Spec.Stream)

	// Get the trench by the label in stream
	selector := client.ObjectKey{
		Namespace: r.ObjectMeta.Namespace,
		Name:      r.ObjectMeta.Labels["trench"],
	}
	trench := &Trench{}
	err := flowClient.Get(context.TODO(), selector, trench)
	if err != nil || trench == nil {
		return nil, fmt.Errorf("unable to find the trench in label, %s cannot be created", r.GroupVersionKind().Kind)
	}

	return nil, r.validateFlow()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	flowlog.Info("validate update", "name", r.Name, "stream", r.Spec.Stream)

	err := r.validateUpdate(old)
	if err != nil {
		return nil, err
	}
	return nil, r.validateFlow()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Flow) ValidateDelete() (admission.Warnings, error) {
	flowlog.Info("validate delete", "name", r.Name, "stream", r.Spec.Stream)

	return nil, nil
}

func (r *Flow) validateFlow() error {
	var allErrs field.ErrorList
	if _, ok := r.ObjectMeta.Labels["trench"]; !ok {
		err := fmt.Errorf("%s must have a trench label", r.GroupVersionKind().Kind)
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata").Child("labels"), r.ObjectMeta.Labels, err.Error()))
	}

	protocols := make(map[string]struct{})
	for _, protocol := range r.Spec.Protocols {
		_, exists := protocols[string(protocol)]
		if exists {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("protocols"), r.Spec.Protocols, "duplicated protocols"))
			break
		}
		protocols[string(protocol)] = struct{}{}
	}

	if n, err := validateSubnets(r.Spec.SourceSubnets); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("source-subnets"), n,
			fmt.Sprintf("source subnet%s", err.Error())))
	}

	if p, err := validatePorts(r.Spec.SourcePorts); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("source-ports"), p,
			fmt.Sprintf("source port%s", err.Error())))
	}

	if p, err := validatePorts(r.Spec.DestinationPorts); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("destination-ports"), p,
			fmt.Sprintf("destination port%s", err.Error())))
	}

	if r.Spec.Priority < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("priority"), r.Spec.Priority,
			"priority must be larger than 0"))
	}

	allErrs = append(allErrs, r.validateVips()...)

	if p, err := validateByteMatches(r.Spec.ByteMatches); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("byte-matches"), p,
			fmt.Sprintf("byte matches%s", err.Error())))
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

func validateByteMatches(byteMatches []string) (string, error) {
	for i, bm := range byteMatches {
		if !reader.ValidByteMatch(bm) {
			return bm, fmt.Errorf("[%d]: byte match wrong format", i)
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

// validateVips performs the complex validation for the Flow's VIPs.
// It checks for potential conflicts with other existing Flows in the same trench.
// The function distinguishes between "hard" conflicts (where both the incoming
// flow and a conflicting flow are found in the cluster with known conduits)
// and "soft" conflicts (where dependency information is not yet available).
// Hard conflicts are treated as user-facing validation errors, while all conflicts
// are logged for debugging and visibility.
// It requires access to a Kubernetes client to list other Flow and Stream objects.
//
// TODO: Because of the design decision not to perform extensive dependency checks,
// admission webhook is not the right place for checking VIP conflicts.
func (r *Flow) validateVips() field.ErrorList {
	var allErrs field.ErrorList
	fl := &FlowList{}
	sel := labels.Set{"trench": r.ObjectMeta.Labels["trench"]}
	err := flowClient.List(context.TODO(), fl, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vips"), r.Spec.Vips,
			fmt.Sprintf("unable to get flows for vip validation: %v", err)))
		return allErrs
	}

	sl := &StreamList{}
	sel = labels.Set{"trench": r.ObjectMeta.Labels["trench"]}
	err = flowClient.List(context.TODO(), sl, &client.ListOptions{
		LabelSelector: sel.AsSelector(),
		Namespace:     r.ObjectMeta.Namespace,
	})
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vips"), r.Spec.Vips,
			fmt.Sprintf("unable to get streams for vip validation: %v", err)))
		return allErrs
	}

	vipConflictDetected := false
	allConflictDetails := []string{}
	hardConflictDetails := []string{}
	streams := map[string]Stream{}
	currentConduit := "" // conduit of the incoming flow
	for _, stream := range sl.Items {
		streams[stream.ObjectMeta.Name] = stream
		if stream.ObjectMeta.Name == r.Spec.Stream {
			currentConduit = stream.Spec.Conduit
		}
	}
	vips := map[string]struct{}{}
	for _, vip := range r.Spec.Vips {
		vips[vip] = struct{}{}
	}
	for _, flow := range fl.Items {
		if flow.ObjectMeta.Name == r.ObjectMeta.Name && flow.ObjectMeta.Namespace == r.ObjectMeta.Namespace {
			continue
		}
		stream, otherStreamExists := streams[flow.Spec.Stream]
		if otherStreamExists && stream.Spec.Conduit == currentConduit {
			continue
		}

		conflictingVips := []string{}
		otherStreamDescription := fmt.Sprintf("stream %q (conduit %q)", flow.Spec.Stream, stream.Spec.Conduit)
		if !otherStreamExists {
			otherStreamDescription = fmt.Sprintf("stream %q (not found)", flow.Spec.Stream)
		}

		for _, vip := range flow.Spec.Vips {
			_, exists := vips[vip]
			if exists {
				vipConflictDetected = true
				conflictingVips = append(conflictingVips, vip)
			}
		}

		if len(conflictingVips) > 0 {
			details := fmt.Sprintf("flow %q (%s) conflicts on vip(s) [%s]",
				flow.ObjectMeta.Name, otherStreamDescription, strings.Join(conflictingVips, ", "))
			allConflictDetails = append(allConflictDetails, details)

			// This indicates a "hard conflict": both the incoming Flow's
			// Stream and the conflicting Flow's Stream were found in the
			// cluster, allowing a direct comparison of their Conduits.
			if currentConduit != "" && otherStreamExists {
				hardConflictDetails = append(hardConflictDetails, details)
			}
		}
	}

	if vipConflictDetected {
		myStreamDescription := fmt.Sprintf("%q (conduit %q)", r.Spec.Stream, currentConduit)
		if currentConduit == "" {
			myStreamDescription = fmt.Sprintf("%q (not found)", r.Spec.Stream)
		}

		// Issue a user-facing error only for "hard conflicts." A hard conflict occurs when
		// both the incoming Flow's stream and the conflicting Flow's stream (and their conduits)
		// are explicitly found in the cluster, indicating a verifiable VIP sharing violation.
		//
		// Background: An early design decision was made that Flow create/update validation should not
		// perform extensive dependency checks towards other associated Custom Resources (like Streams)
		// for user convenience during complex deployments. Therefore, "soft conflicts" (e.g., those
		// involving streams that are not yet found or are in an unknown state) are not treated as strict
		// errors here, preventing spurious validation failures due to transient unsynchronized states
		// between Flow and Stream objects.
		if len(hardConflictDetails) > 0 {
			combinedDetailForUser := fmt.Sprintf("a vip cannot be shared between 2 conduits in this version: %s, %s",
				fmt.Sprintf("incoming flow %q (%s)", r.Name, myStreamDescription), strings.Join(hardConflictDetails, ", "))
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("vips"), r.Spec.Vips, combinedDetailForUser))
		}

		logPrefix := ""
		if len(hardConflictDetails) == 0 {
			logPrefix = "possible "
		}

		// Log all conflicts (hard and soft) for full visibility and debugging
		flowlog.Info(fmt.Sprintf("validate flow: %svip conflict detected", logPrefix),
			"flow", r.Name,
			"stream", myStreamDescription,
			"details", strings.Join(allConflictDetails, "; "))
	}
	return allErrs
}

func (r *Flow) validateUpdate(oldObj runtime.Object) error {
	old, ok := oldObj.(*Flow)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a %s got a %T", r.GroupVersionKind().Kind, old))
	}
	trenchNew := r.ObjectMeta.Labels["trench"]
	trenchOld := old.ObjectMeta.Labels["trench"]
	if trenchNew != trenchOld {
		return apierrors.NewForbidden(r.GroupResource(),
			r.Name, field.Forbidden(field.NewPath("metadata", "labels", "trench"), "update on trench label is forbidden"))
	}
	return nil
}
