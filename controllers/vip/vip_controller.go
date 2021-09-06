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
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
)

// VipReconciler reconciles a Vip object
type VipReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips/finalizers,verbs=update

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

	vip := &meridiov1alpha1.Vip{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, vip)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err != nil {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// clear vip status
	currentVip := vip.DeepCopy()
	vip = setVipStatus(vip, meridiov1alpha1.NoPhase, "")

	trench, attr, err := validateVip(executor, vip)
	if err != nil {
		return ctrl.Result{}, err
	}

	// actions to update vip
	if attr != nil {
		err = executor.SetOwnerReference(vip, trench, attr)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	actions := getVipActions(executor, vip, currentVip)
	err = executor.RunAll(actions)

	return ctrl.Result{}, err
}

func validateVip(e *common.Executor, vip *meridiov1alpha1.Vip) (*meridiov1alpha1.Trench, *meridiov1alpha1.Attractor, error) {
	// Get the trench by the label in vip
	selector := client.ObjectKey{
		Namespace: vip.ObjectMeta.Namespace,
		Name:      vip.ObjectMeta.Labels["trench"],
	}
	trench, err := common.GetTrenchbySelector(e, selector)
	if err != nil {
		// set vip status to rejected if trench is not found
		if apierrors.IsNotFound(err) {
			setVipStatus(vip,
				meridiov1alpha1.Disengaged,
				"labeled trench not found")
			return nil, nil, nil
		} else {
			return nil, nil, err
		}
	}

	attrname, ok := vip.ObjectMeta.Labels["attractor"]
	if !ok {
		setVipStatus(vip,
			meridiov1alpha1.Disengaged,
			"labeled trench not found")
		return nil, nil, nil
	}
	selector = client.ObjectKey{
		Namespace: vip.ObjectMeta.Namespace,
		Name:      attrname,
	}
	attr := &meridiov1alpha1.Attractor{}
	if err := e.GetObject(selector, attr); err != nil {
		if apierrors.IsNotFound(err) {
			msg := "labeled attractor not found"
			vip.Status.Status = meridiov1alpha1.Disengaged
			vip.Status.Message = msg
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if attr.ObjectMeta.Labels["trench"] != vip.ObjectMeta.Labels["trench"] {
		msg := "attractor and trench label mismatch"
		vip.Status.Status = meridiov1alpha1.Disengaged
		vip.Status.Message = msg
		return nil, nil, nil
	}

	setVipStatus(vip, meridiov1alpha1.Engaged, "")
	return trench, attr, nil
}

func setVipStatus(vip *meridiov1alpha1.Vip, status meridiov1alpha1.ConfigStatus, msg string) *meridiov1alpha1.Vip {
	vip.Status.Status = status
	vip.Status.Message = msg
	return vip
}

func getVipActions(e *common.Executor, new, old *meridiov1alpha1.Vip) []common.Action {
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

// SetupWithManager sets up the controller with the Manager.
func (r *VipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Vip{}).
		Complete(r)
}
