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

package attractor

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	common "github.com/nordix/meridio-operator/controllers/common"
)

// AttractorReconciler reconciles a Attractor object
type AttractorReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Log        logr.Logger
	PdbVersion string
}

//+kubebuilder:rbac:groups=meridio.nordix.org,namespace=system,resources=attractors,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=apps,resources=deployments,namespace=system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,namespace=system,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *AttractorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	attr := &meridiov1alpha1.Attractor{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, attr)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	currentAttr := attr.DeepCopy()
	attr.Status = meridiov1alpha1.AttractorStatus{}

	selector := client.ObjectKey{
		Namespace: attr.ObjectMeta.Namespace,
		Name:      attr.ObjectMeta.Labels["trench"],
	}
	trench, _ := common.GetTrenchBySelector(executor, selector)

	if trench != nil {
		// update attractor
		executor.SetOwnerReference(attr, trench)
		// create/update stateless-lb-frontend & nse-vlan deployment
		executor.SetOwner(attr)

		nse, err := NewNSE(executor, attr, trench)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = nse.getAction()
		if err != nil {
			return ctrl.Result{}, err
		}

		lb, err := NewLoadBalancer(executor, attr, trench)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = lb.getAction()
		if err != nil {
			return ctrl.Result{}, err
		}

		lbPDB, err := NewLoadBalancerPDB(executor, attr, trench, r.PdbVersion)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = lbPDB.getAction()
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		return ctrl.Result{}, fmt.Errorf("unable to get trench for attractor %s", req.NamespacedName)
	}

	getAttractorActions(executor, attr, currentAttr)
	err = executor.RunActions()
	return ctrl.Result{}, err
}

func getAttractorActions(e *common.Executor, new, old *meridiov1alpha1.Attractor) {
	if !equality.Semantic.DeepEqual(new.Status, old.Status) {
		e.AddUpdateStatusAction(new)
	}
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		e.AddUpdateAction(new)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AttractorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.PdbVersion, err = common.GetPodDisruptionBudgetVersion(r.Client)
	if err != nil {
		return err
	}
	var podDisruptionBudget client.Object
	podDisruptionBudget = &policyv1.PodDisruptionBudget{}
	if r.PdbVersion == policyv1beta1.SchemeGroupVersion.Version {
		podDisruptionBudget = &policyv1beta1.PodDisruptionBudget{}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Attractor{}).
		Owns(&appsv1.Deployment{}).
		Owns(podDisruptionBudget).
		Complete(r)
}
