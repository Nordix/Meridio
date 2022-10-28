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

package trench

import (
	"context"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	"github.com/nordix/meridio/pkg/controllers/common"
)

// TrenchReconciler reconciles a Trench object
type TrenchReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,namespace=system,resources=trenches,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=apps,resources=deployments,namespace=system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,namespace=system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,namespace=system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,namespace=system,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,namespace=system,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Trench object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *TrenchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("trench", req.NamespacedName)

	trench := &meridiov1alpha1.Trench{}
	err := r.Get(ctx, req.NamespacedName, trench)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	executor := common.NewExecutor(r.Scheme, r.Client, ctx, trench, r.Log)
	meridio, err := NewMeridio(executor, trench)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = meridio.ReconcileAll()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = executor.RunActions()
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrenchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Trench{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Attractor{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of Attractors, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Vip{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of Vips, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Gateway{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of gateways, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Conduit{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of Conduits, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Stream{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of Streams, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &meridiov1alpha1.Flow{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of Flow, so here uses Watches with IsController: false
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestForOwner{OwnerType: &meridiov1alpha1.Trench{}, IsController: false},
		). // Trenches are not the controllers of configmaps, so here uses Watches with IsController: false
		Complete(r)
}
