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

package conduit

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
)

// ConduitReconciler reconciles a Conduit object
type ConduitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=conduits,namespace=system,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=apps,resources=daemonsets,namespace=system,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Conduit object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ConduitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	conduit := &meridiov1.Conduit{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, conduit)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get conduit (%s) in conduit controller: %w", req.Name, err)
	}
	// clear conduit status
	currentConduit := conduit.DeepCopy()

	// Get the trench by the label in conduit
	selector := client.ObjectKey{
		Namespace: conduit.ObjectMeta.Namespace,
		Name:      conduit.ObjectMeta.Labels["trench"],
	}
	trench, _ := common.GetTrenchBySelector(executor, selector)

	// actions to update conduit
	if trench != nil {
		err = executor.SetOwnerReference(conduit, trench)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set owner reference (%s) in conduit controller: %w", req.Name, err)
		}
		// create/update stateless-lb-frontend & nse-vlan deployment
		executor.SetOwner(conduit)

		proxy, err := NewProxy(executor, trench, conduit)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = proxy.getAction()
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{}, fmt.Errorf("unable to get trench for conduit %s", req.NamespacedName)
	}

	getConduitActions(executor, conduit, currentConduit)
	err = executor.RunActions()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to run actions (%s) in conduit controller: %w", req.Name, err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConduitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1.Conduit{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build conduit controller: %w", err)
	}

	return nil
}

func getConduitActions(executor *common.Executor, new, old *meridiov1.Conduit) {
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		executor.AddUpdateAction(new)
	}
}
