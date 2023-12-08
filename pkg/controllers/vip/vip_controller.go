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

package vip

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
)

// VipReconciler reconciles a Vip object
type VipReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips,namespace=system,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Vip object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *VipReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("vip", req.NamespacedName)

	vip := &meridiov1.Vip{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, vip)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get vip (%s) in vip controller: %w", req.Name, err)
	}

	// save the current vip status
	currentVip := vip.DeepCopy()

	// Get the trench by the label in vip
	selector := client.ObjectKey{
		Namespace: vip.ObjectMeta.Namespace,
		Name:      vip.ObjectMeta.Labels["trench"],
	}
	trench, _ := common.GetTrenchBySelector(executor, selector)

	// actions to update vip
	if trench != nil {
		err = executor.SetOwnerReference(vip, trench)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set owner reference (%s) in vip controller: %w", req.Name, err)
		}
	} else {
		return ctrl.Result{}, fmt.Errorf("unable to get trench for vip %s", req.NamespacedName)
	}

	getVipActions(executor, vip, currentVip)
	err = executor.RunActions()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to run actions (%s) in vip controller: %w", req.Name, err)
	}

	return ctrl.Result{}, nil
}

func getVipActions(executor *common.Executor, new, old *meridiov1.Vip) {
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		executor.AddUpdateAction(new)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1.Vip{}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build vip controller: %w", err)
	}

	return nil
}
