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
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
)

// ConduitReconciler reconciles a Conduit object
type ConduitReconciler struct {
	client.Client
	APIReader        client.Reader // reader contacting the API server directly bypassing the local cache
	Log              logr.Logger
	Scheme           *runtime.Scheme
	updateSyncGroups sync.Map // map stores Update Sync Group annotation values for each conduit Custom Resource object part of one
	updateLocks      sync.Map // map stores locks for Upgrade Sync Groups
}

// Note: In memory lock. Not effective if operator crashes during ongoing
// updates, but that would be considered a bug to be fixed anyways.
// (Compared to the complexity of a ConfigMap based locking this provides
// good enough protection and is probably much faster.)
type updateLock struct {
	mutex      *sync.Mutex // resolve race conditions in case of parallel sync.Map calls with the same key (e.g.: Load, Store, Delete etc.)
	owner      string
	acquiredAt time.Time
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
	logger := log.FromContext(ctx)

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
	updateSyncGroup := getUpdateSyncGroup(currentConduit) // Check if conduit belongs to an update sync group
	logger = logger.WithValues("updateSyncGroup", updateSyncGroup)
	r.manageUpdateSyncGroupChange(updateSyncGroup, req.NamespacedName.String()) // Check if update sync group has changed

	// Get the trench by the label in conduit
	selector := client.ObjectKey{
		Namespace: conduit.ObjectMeta.Namespace,
		Name:      conduit.ObjectMeta.Labels["trench"],
	}
	trench, _ := common.GetTrenchBySelector(executor, selector)

	// actions to update conduit
	if trench == nil {
		return ctrl.Result{}, fmt.Errorf("unable to get trench for conduit %s", req.NamespacedName)
	}
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
	proxyUpdateAction, err := proxy.getAction()
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle update locking in case conduit belongs to an update sync group,
	// but try not to block/delay execution of initial proxy create action.
	// Note: Initial proxy create action can add the daemonset, allowing PODs
	// to be created etc. But after that any subsequent reconcile actions will
	// be considered updates due to the existing daemonset.
	if updateSyncGroup != "" {
		if proxyUpdateAction {
			if !r.acquireLock(updateSyncGroup, req.NamespacedName.String()) {
				logger.Info("Update locked by another conduit, requeueing")
				// TODO: consider making delay configurable
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			logger.Info("Lock acquired")
		}
		if r.hasLock(updateSyncGroup, req.NamespacedName.String()) {
			defer func() {
				// Check if proxy daemonset along with its PODs have been updated
				// after pushing the actions to the API server. If so release the lock.
				// Otherwise, wait for the next Reconcile call.
				// Note: Must fetch up to date version of proxy workload(s), instead of
				// an old one from the cache.
				ok, err := r.isProxyUpdateReady(common.NewReader(r.Scheme, r.APIReader, ctx, r.Log), proxy)
				if ok {
					logger.Info("Updates finished, releasing lock")
					r.releaseLock(updateSyncGroup, req.NamespacedName.String())
				} else {
					logger.Info("Updates ongoing, not releasing lock", "details", err)
				}
			}()
		}
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
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				// Handle conduit Custom Resource deletion from locking perspective
				if _, ok := e.Object.(*meridiov1.Conduit); ok {
					r.cleanupLock(e.Object.GetName(), e.Object.GetNamespace())
				}
				return true
			},
		}).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build conduit controller: %w", err)
	}

	return nil
}

// Try to acquire update sync group lock for owner
func (r *ConduitReconciler) acquireLock(updateSyncGroup, owner string) bool {
	lock, loaded := r.updateLocks.LoadOrStore(updateSyncGroup, &updateLock{mutex: &sync.Mutex{}})
	ul := lock.(*updateLock)
	ul.mutex.Lock()
	defer ul.mutex.Unlock()

	logger := r.Log.WithName("acquireLock").WithValues("updateSyncGroup", updateSyncGroup, "caller", owner)
	if ul.owner != "" && ul.owner != owner {
		logger.V(1).Info("Lock in use", "owner", ul.owner)
		return false
	}
	if !loaded {
		// We just created the lock
		ul.owner = owner
		ul.acquiredAt = time.Now()
		logger.Info("Lock created", "owner", ul.owner)
	}
	return true
}

// Check if owner has locked the update sync group
func (r *ConduitReconciler) hasLock(updateSyncGroup, owner string) bool {
	logger := r.Log.WithName("hasLock").WithValues("updateSyncGroup", updateSyncGroup, "caller", owner)
	lock, ok := r.updateLocks.Load(updateSyncGroup)
	if !ok {
		logger.V(1).Info("No lock found")
		return false
	}
	ul := lock.(*updateLock)
	ul.mutex.Lock()
	defer ul.mutex.Unlock()
	logger.V(1).Info("Verify ownership", "owner", ul.owner, "acquiredAt", ul.acquiredAt)
	return ul.owner != "" && ul.owner == owner
}

// Release lock for update sync group if held by owner
func (r *ConduitReconciler) releaseLock(updateSyncGroup, owner string) {
	lock, ok := r.updateLocks.Load(updateSyncGroup)
	if !ok {
		return
	}
	ul := lock.(*updateLock)
	ul.mutex.Lock()
	defer ul.mutex.Unlock()
	if ul.owner == owner {
		// Note: owner is not cleared for a reason, thus even if the lock was loaded in
		// the meantime by someone else it should not result in an inconsistent state,
		// since the lock should not be touached if the owner is different.
		r.Log.WithName("releaseLock").Info("Released", "updateSyncGroup", updateSyncGroup,
			"owner", owner, "acquiredAt", ul.acquiredAt)
		r.updateLocks.Delete(updateSyncGroup)
	}
}

// Store new updateSyncGroup value for the conduit.
// Also, upon change in update sync group, previously held lock if any must be released.
func (r *ConduitReconciler) manageUpdateSyncGroupChange(updateSyncGroup, conduitName string) {
	prevUpdateSyncGroup, ok := r.updateSyncGroups.Load(conduitName)

	hasUpdateSyncGroupChanged := func() bool {
		if ok && updateSyncGroup == prevUpdateSyncGroup { // same as the stored group
			return false
		}
		if !ok && updateSyncGroup == "" { // new is empty string and no previous value
			return false
		}
		return true
	}

	syncUpdateSyncGroupStorage := func() {
		if updateSyncGroup != "" { // save new value
			r.updateSyncGroups.Store(conduitName, updateSyncGroup)
		} else { // empty string new value, remove old
			r.updateSyncGroups.Delete(conduitName)
		}
	}

	if !hasUpdateSyncGroupChanged() {
		return
	}

	r.Log.WithName("manageUpdateSyncGroupChange").Info("Change observed", "conduit", conduitName,
		"updateSyncGroup", updateSyncGroup, "prevUpdateSyncGroup", prevUpdateSyncGroup)

	if ok { // release former lock if any
		r.releaseLock(prevUpdateSyncGroup.(string), conduitName)
	}
	syncUpdateSyncGroupStorage()
}

// Find update sync group and release lock if held by the conduit
func (r *ConduitReconciler) cleanupLock(name, namespace string) {
	key := types.NamespacedName{Name: name, Namespace: namespace}.String()
	logger := r.Log.WithName("cleanupLock").WithValues("conduit", key)
	logger.Info("Clean up lock if any")
	updateSyncGroup, loaded := r.updateSyncGroups.LoadAndDelete(key)
	if !loaded {
		logger.V(1).Info("Not part of any updateSyncGroup")
		return
	}
	r.releaseLock(updateSyncGroup.(string), key)
}

func (r *ConduitReconciler) isProxyUpdateReady(reader *common.Reader, proxy *Proxy) (bool, error) {
	return proxy.isReady(reader)
}

func getConduitActions(executor *common.Executor, new, old *meridiov1.Conduit) {
	if !equality.Semantic.DeepEqual(new.ObjectMeta, old.ObjectMeta) {
		executor.AddUpdateAction(new)
	}
}

func getUpdateSyncGroup(conduit *meridiov1.Conduit) string {
	updateSyncGroupAnnotation := common.GetConduitUpdateSyncGroupKey() // allows customization of annotation key
	if val, ok := conduit.GetAnnotations()[updateSyncGroupAnnotation]; ok {
		return val
	}
	return ""
}
