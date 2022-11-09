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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	"github.com/nordix/meridio/pkg/controllers/common"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FlowReconciler reconciles a Flow object
type FlowReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=flows,namespace=system,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Flow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *FlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	flow := &meridiov1alpha1.Flow{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, flow)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// clear flow status
	currentFlow := flow.DeepCopy()

	selector := client.ObjectKey{
		Namespace: flow.ObjectMeta.Namespace,
		Name:      flow.ObjectMeta.Labels["trench"],
	}
	trench, _ := common.GetTrenchBySelector(executor, selector)

	// actions to update flow
	if trench != nil {
		err = executor.SetOwnerReference(flow, trench)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{}, fmt.Errorf("unable to get trench for stream %s", req.NamespacedName)
	}

	getFlowActions(executor, flow, currentFlow)
	err = executor.RunActions()

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Flow{}).
		Complete(r)
}

func getFlowActions(executor *common.Executor, new, old *meridiov1alpha1.Flow) []common.Action {
	var actions []common.Action
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		executor.AddUpdateAction(new)
	}
	return actions
}
