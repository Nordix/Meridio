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
	"net"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
	"github.com/nordix/meridio-operator/controllers/common"
)

// VipReconciler reconciles a Vip object
type VipReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	TrenchVip map[string]map[string]map[string]*net.IPNet //namespace->trench->vip name->vip address
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips,verbs=get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=vips/finalizers,verbs=update
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=trenches,verbs=get
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

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

	configmap := &ConfigMap{}
	vip := &meridiov1alpha1.Vip{}
	executor := common.NewExecutor(r.Scheme, r.Client, ctx, nil, r.Log)

	err := r.Get(ctx, req.NamespacedName, vip)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.TrenchVip, err = configmap.deleteKey(executor, req.Namespace, req.Name, r.TrenchVip)
			if err != nil {
				return ctrl.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// clear vip status
	vip = setVipStatus(vip, meridiov1alpha1.BaseStatus.NoPhase, "")

	// Get the trench by the label in vip
	selector := client.ObjectKey{
		Namespace: vip.ObjectMeta.Namespace,
		Name:      vip.ObjectMeta.Labels["trench"],
	}
	trench, err := common.GetTrenchbySelector(executor, selector)
	if err != nil {
		// set vip status to rejected if trench is not found
		if apierrors.IsNotFound(err) {
			vip = setVipStatus(vip,
				meridiov1alpha1.ConfigStatus.Rejected,
				fmt.Sprintf("trench %s must be created first", vip.ObjectMeta.Labels["trench"]))
		} else {
			return ctrl.Result{}, err
		}
	}

	if vip.Status.Status != meridiov1alpha1.ConfigStatus.Rejected {
		// record trench and ns in a map
		r.addNsTrenchToMap(trench)
		executor.SetOwner(trench)
		// validate overlapping, set vip status to rejected if there is overlapping
		_, vIPNets, _ := net.ParseCIDR(vip.Spec.Address)
		if err := vipsOverlap(r.TrenchVip[trench.ObjectMeta.Namespace][trench.ObjectMeta.Name], vIPNets, vip.ObjectMeta.Name); err != nil {
			vip = setVipStatus(vip,
				meridiov1alpha1.ConfigStatus.Rejected,
				fmt.Sprintf("validation error: %s", err))
		} else {
			// only add vip which is not rejected to the map
			r.TrenchVip[vip.ObjectMeta.Namespace][trench.ObjectMeta.Name][vip.ObjectMeta.Name] = vIPNets
		}
	}

	if vip.Status.Status != meridiov1alpha1.ConfigStatus.Rejected {
		vip = setVipStatus(vip, meridiov1alpha1.ConfigStatus.Accepted, "")
	}
	// actions to update vip
	actions := getVipActions(executor, vip)
	err = executor.RunAll(actions)
	if err != nil {
		return ctrl.Result{}, err
	}
	if vip.Status.Status == meridiov1alpha1.ConfigStatus.Rejected {
		return ctrl.Result{}, nil
	}
	// action to update update/create configmap
	action, err := configmap.getAction(executor, r.TrenchVip[vip.ObjectMeta.Namespace][trench.ObjectMeta.Name], vip)
	if err != nil {
		return ctrl.Result{}, err
	}
	if action != nil {
		executor.RunAll(append(actions, action))
	}

	return ctrl.Result{}, nil
}

func setVipStatus(vip *meridiov1alpha1.Vip, status string, msg string) *meridiov1alpha1.Vip {
	vip.Status.Status = status
	vip.Status.Message = msg
	return vip
}

func vipsOverlap(allVips map[string]*net.IPNet, vaddr *net.IPNet, skipName string) error {
	for vipName, addr := range allVips {
		if vipName != skipName {
			if cidrsOverlap(addr, vaddr) {
				return fmt.Errorf("vip %s overlapping", vipName)
			}
		}
	}
	return nil
}

func cidrsOverlap(a, b *net.IPNet) bool {
	return cidrContainsCIDR(a, b) || cidrContainsCIDR(b, a)
}

func cidrContainsCIDR(outer, inner *net.IPNet) bool {
	ol, _ := outer.Mask.Size()
	il, _ := inner.Mask.Size()
	if ol == il && outer.IP.Equal(inner.IP) {
		return true
	}
	if ol < il && outer.Contains(inner.IP) {
		return true
	}
	return false
}

func (r *VipReconciler) addNsTrenchToMap(trench *meridiov1alpha1.Trench) {
	if _, ok := r.TrenchVip[trench.ObjectMeta.Namespace]; !ok {
		r.TrenchVip[trench.ObjectMeta.Namespace] = make(map[string]map[string]*net.IPNet)
	}
	if _, ok := r.TrenchVip[trench.ObjectMeta.Namespace][trench.ObjectMeta.Name]; !ok {
		r.TrenchVip[trench.ObjectMeta.Namespace][trench.ObjectMeta.Name] = make(map[string]*net.IPNet)
	}
}

func getVipActions(e *common.Executor, vip *meridiov1alpha1.Vip) []common.Action {
	var actions []common.Action
	// set the status for the vip
	vipnsname := fmt.Sprintf("%s/%s", vip.GetNamespace(), vip.GetName())
	// if vip is rejected due to trench not found, update the status only
	actions = append(actions, common.NewUpdateStatusAction(vip, fmt.Sprintf("update vip %s status: %v", vipnsname, vip.Status.Status)))
	if e.GetOwner().(*meridiov1alpha1.Trench) == nil {
		return actions
	}
	// if vip is rejected due to overlapping address, also
	actions = append(actions, common.NewUpdateAction(vip, fmt.Sprintf("update vip %s ownerReference", vipnsname)))
	return actions
}

// SetupWithManager sets up the controller with the Manager.
func (r *VipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Vip{}).
		Complete(r)
}
