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

package gateway

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,namespace=system,resources=gateways,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=meridio.nordix.org,namespace=system,resources=gateways/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meridio.nordix.org,namespace=system,resources=gateways/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	gw := &meridiov1alpha1.Gateway{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, gw)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	cgw := gw.DeepCopy()
	gw.Status = meridiov1alpha1.GatewayStatus{}
	attr, err := validateGateway(executor, gw)
	if err != nil {
		return ctrl.Result{}, err
	}
	executor.SetOwnerReference(gw, attr)
	actions := getGatewayActions(executor, gw, cgw)
	err = executor.RunAll(actions)
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Gateway{}).
		Complete(r)
}

func validateGateway(e *common.Executor, gw *meridiov1alpha1.Gateway) (*meridiov1alpha1.Attractor, error) {
	// get the attractor by gateway label
	selector := client.ObjectKey{
		Namespace: gw.ObjectMeta.Namespace,
		Name:      gw.ObjectMeta.Labels["attractor"],
	}
	attr := &meridiov1alpha1.Attractor{}
	if err := e.GetObject(selector, attr); err != nil {
		if apierrors.IsNotFound(err) {
			msg := "labeled attractor not found"
			gw.Status.Status = meridiov1alpha1.Disengaged
			gw.Status.Message = msg
			return nil, nil
		}
		return nil, err
	}
	gw.Status.Status = meridiov1alpha1.Engaged
	return attr, nil
}

func getGatewayActions(e *common.Executor, new, old *meridiov1alpha1.Gateway) []common.Action {
	var actions []common.Action
	// set the status for the vip
	nsname := common.NsName(new.ObjectMeta)
	if !equality.Semantic.DeepEqual(new.Status, old.Status) {
		actions = append(actions, common.NewUpdateStatusAction(new, fmt.Sprintf("update %s status: %v", nsname, new.Status.Status)))
	}
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		actions = append(actions, common.NewUpdateAction(new, fmt.Sprintf("update %s ownerReference", nsname)))
	}
	return actions
}
