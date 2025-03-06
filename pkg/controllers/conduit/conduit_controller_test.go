/*
Copyright (c) 2025 OpenInfra Foundation Europe

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

package conduit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	meridiov1 "github.com/nordix/meridio/api/v1"
	"github.com/nordix/meridio/pkg/controllers/common"
	"github.com/nordix/meridio/pkg/controllers/conduit"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Note: The fake client simulates a Kubernetes API server using an in-memory
// object store. Create and Update etc. operations are applied immediately. However,
// it doesn't replicate all API server functionalities. Notably, metadata.generation,
// DaemonSet Status subresources, and automatic Reconcile calls are not automatically
// updated/triggered.

var updateSyncGroupKey string = common.GetConduitUpdateSyncGroupKey()

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = meridiov1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	return scheme
}

func newConduit(name, trench, syncGroup string) *meridiov1.Conduit {
	conduit := &meridiov1.Conduit{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"trench": trench,
			},
		},
	}
	if syncGroup != "" {
		conduit.Annotations = map[string]string{updateSyncGroupKey: syncGroup}
	}
	return conduit
}

// newProxyDaemonSet returns a basic proxy DaemonSet with a simplified specification,
// intentionally different from a fully configured DaemonSet, that the Conduit Reconciler
// aims to achieve.
// The 'desiredNumberScheduled' parameter allows control over the DaemonSet's status,
// enabling simulation of various reconciliation progress states such as slow updates.
func newProxyDaemonSet(conduit *meridiov1.Conduit, desiredNumberScheduled int32) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ProxyDeploymentName(conduit),
			Namespace: conduit.GetNamespace(),
			Labels: map[string]string{
				"app": "proxy",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "proxy"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "proxy"},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: desiredNumberScheduled,
		},
	}
	return ds
}

func setupReconciler(t *testing.T, objects ...client.Object) (*conduit.ConduitReconciler, client.Client) {
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objects...).Build()
	logger := testr.New(t)

	return &conduit.ConduitReconciler{
		Client:           fakeClient,
		APIReader:        fakeClient,
		Scheme:           newScheme(),
		Log:              logger.WithName("test"),
		ProxyModelLoader: &TestProxyModelLoader{}, // Test Loader bypasses normal operation by not reading the proxy model from template file
	}, fakeClient
}

func assertLockStatus(t *testing.T, reconciler *conduit.ConduitReconciler, conduit *meridiov1.Conduit, expectLock bool) {
	hasLock := reconciler.HasLock(
		conduit.GetAnnotations()[common.GetConduitUpdateSyncGroupKey()],
		types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}.String(),
	)
	assert.Equal(t, expectLock, hasLock)
}

func assertDaemonSet(t *testing.T, fakeClient client.Client, conduit *meridiov1.Conduit) *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{}
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: common.ProxyDeploymentName(conduit), Namespace: conduit.Namespace}, ds)
	assert.NoError(t, err)
	assert.NotNil(t, ds.Spec)
	return ds
}

func assertNoDaemonSet(t *testing.T, fakeClient client.Client, conduit *meridiov1.Conduit) {
	ds := &appsv1.DaemonSet{}
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: common.ProxyDeploymentName(conduit), Namespace: conduit.Namespace}, ds)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func assertConduit(t *testing.T, fakeClient client.Client, conduit *meridiov1.Conduit) *meridiov1.Conduit {
	c := &meridiov1.Conduit{}
	err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}, c)
	assert.NoError(t, err)
	return c
}

func TestConduitReconciler_Reconcile(t *testing.T) {
	t.Run("CreateConduits_NoSyncGroup_CreatesProxyDaemonSets", func(t *testing.T) {
		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", ""),
			newConduit("test-conduit-B", "test-trench", ""),
		}
		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c)
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		for _, c := range conduits {
			assertNoDaemonSet(t, fakeClient, c)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}}
			result, err := reconciler.Reconcile(context.Background(), request)
			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			_ = assertDaemonSet(t, fakeClient, c)
		}
		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_NoSyncGroup_UpdatesProxyDaemonSets", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", ""),
			newConduit("test-conduit-B", "test-trench", ""),
		}
		var objects []client.Object
		// Initialize fake client with daemonsets to mimic update
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 0))
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		for _, c := range conduits {
			cs := assertDaemonSet(t, fakeClient, c)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}}
			result, err := reconciler.Reconcile(context.Background(), request)
			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			ds := assertDaemonSet(t, fakeClient, c)
			assert.False(t, equality.Semantic.DeepEqual(ds.Spec, cs.Spec)) // ds update executed
		}

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("CreateConduits_SharedSyncGroup_CreatesProxyDaemonSets", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}
		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c)
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		for _, c := range conduits {
			assertNoDaemonSet(t, fakeClient, c)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}}
			result, err := reconciler.Reconcile(context.Background(), request)
			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			cond := assertConduit(t, fakeClient, c)
			assert.Equal(t, "test-group", conduit.GetUpdateSyncGroup(cond))

			_ = assertDaemonSet(t, fakeClient, c)
		}
		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_SharedSyncGroup_UpdatesProxyDaemonSets", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}
		var objects []client.Object
		// Initialize fake client with daemonsets to mimic update
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 0))
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		for _, c := range conduits {
			cs := assertDaemonSet(t, fakeClient, c)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}}
			result, err := reconciler.Reconcile(context.Background(), request)
			assert.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)

			cond := assertConduit(t, fakeClient, c)
			assert.Equal(t, "test-group", conduit.GetUpdateSyncGroup(cond))

			ds := assertDaemonSet(t, fakeClient, c)

			assert.False(t, equality.Semantic.DeepEqual(ds.Spec, cs.Spec)) // ds update executed
		}

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_SharedSyncGroup_SlowProgress", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}

		var objects []client.Object
		objects = append(objects, conduits[0], newProxyDaemonSet(conduits[0], 1)) // Slow conduit
		objects = append(objects, conduits[1], newProxyDaemonSet(conduits[1], 0)) // Normal conduit
		reconciler, fakeClient := setupReconciler(t, objects...)

		// First reconcile - slow conduit gets the lock
		// But update cannot conclude because fake client is a mock and lacks
		// api server functionality to auto-update DaemonSet Status subresource.
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err := reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], true)
		_ = assertDaemonSet(t, fakeClient, conduits[0])

		// Second reconcile - normal conduit must wait for the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Greater(t, result.RequeueAfter, time.Duration(0))
		assertLockStatus(t, reconciler, conduits[1], false)
		_ = assertDaemonSet(t, fakeClient, conduits[1])

		// Simulate slow conduit's DaemonSet reaching ready state
		ds := &appsv1.DaemonSet{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: common.ProxyDeploymentName(conduits[0]), Namespace: conduits[0].Namespace}, ds)
		assert.NoError(t, err)
		ds.Status.DesiredNumberScheduled = 0
		err = fakeClient.Status().Update(context.Background(), ds)
		assert.NoError(t, err)

		// Third reconcile - slow conduit releases the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], false)

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_SharedSyncGroup_DeleteStalledConduit", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}

		var objects []client.Object
		objects = append(objects, conduits[0], newProxyDaemonSet(conduits[0], 1)) // Slow conduit
		objects = append(objects, conduits[1], newProxyDaemonSet(conduits[1], 0)) // Normal conduit
		reconciler, fakeClient := setupReconciler(t, objects...)

		// First reconcile - slow conduit gets the lock
		// But update cannot conclude because fake client is a mock and lacks
		// api server functionality to auto-update DaemonSet Status subresource.
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err := reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], true)
		_ = assertDaemonSet(t, fakeClient, conduits[0])

		// Second reconcile - normal conduit must wait for the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Greater(t, result.RequeueAfter, time.Duration(0))
		assertLockStatus(t, reconciler, conduits[1], false)
		_ = assertDaemonSet(t, fakeClient, conduits[1])

		// Simulate deletion of slow conduit holding the lock. Fake client lacks
		// automatic deletion handling, so manually trigger HandleConduitDeletion.
		c := &meridiov1.Conduit{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}, c)
		assert.NoError(t, err)
		err = fakeClient.Delete(context.Background(), c)
		assert.NoError(t, err)
		reconciler.HandleConduitDeletion(c.Name, c.Namespace) // releases lock and removes the conduit to update sync group mapping
		assertLockStatus(t, reconciler, conduits[0], false)
		assert.Equal(t, len(conduits)-1, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

		// Third reconcile - normal conduit can execute the update
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[1], false)

		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

	})

	t.Run("UpdateConduits_SharedSyncGroup_RemoveStalledConduitAnnotation", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}

		var objects []client.Object
		objects = append(objects, conduits[0], newProxyDaemonSet(conduits[0], 1)) // Slow conduit
		objects = append(objects, conduits[1], newProxyDaemonSet(conduits[1], 0)) // Normal conduit
		reconciler, fakeClient := setupReconciler(t, objects...)

		// First reconcile - slow conduit gets the lock
		// But update cannot conclude because fake client is a mock and lacks
		// api server functionality to auto-update DaemonSet Status subresource.
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err := reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], true)
		_ = assertDaemonSet(t, fakeClient, conduits[0])

		// Second reconcile - normal conduit must wait for the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Greater(t, result.RequeueAfter, time.Duration(0))
		assertLockStatus(t, reconciler, conduits[1], false)
		_ = assertDaemonSet(t, fakeClient, conduits[1])

		// Modify slow conduit's annotation to remove the update sync group
		c := &meridiov1.Conduit{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}, c)
		assert.NoError(t, err)
		c.Annotations = make(map[string]string) // Remove all annotations, including update sync group
		err = fakeClient.Update(context.Background(), c)
		assert.NoError(t, err)

		// Third reconcile - slow conduit releases the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], false)
		assert.Equal(t, len(conduits)-1, len(reconciler.GetUpdateSyncGroups())) // Slow conduit's update sync group was removed
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

		// Fourth reconcile - normal conduit can execute the update
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[1], false)

		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_SharedSyncGroup_ChangeStalledConduitAnnotation", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group"),
			newConduit("test-conduit-B", "test-trench", "test-group"),
		}

		var objects []client.Object
		objects = append(objects, conduits[0], newProxyDaemonSet(conduits[0], 1)) // Slow conduit
		objects = append(objects, conduits[1], newProxyDaemonSet(conduits[1], 0)) // Normal conduit
		reconciler, fakeClient := setupReconciler(t, objects...)

		// First reconcile - slow conduit gets the lock
		// But update cannot conclude because fake client is a mock and lacks
		// api server functionality to auto-update DaemonSet Status subresource.
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err := reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], true)
		_ = assertDaemonSet(t, fakeClient, conduits[0])

		// Second reconcile - normal conduit must wait for the lock
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Greater(t, result.RequeueAfter, time.Duration(0))
		assertLockStatus(t, reconciler, conduits[1], false)
		_ = assertDaemonSet(t, fakeClient, conduits[1])

		// Modify slow conduit's annotation to change its update sync group
		c := &meridiov1.Conduit{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}, c)
		assert.NoError(t, err)
		c.Annotations[updateSyncGroupKey] = "new-test-group" // change the sync group
		err = fakeClient.Update(context.Background(), c)
		assert.NoError(t, err)

		// Third reconcile - slow conduit releases its former update sync group's lock
		// Lock for the new update sync group is not acquired if no change is observed
		// in proxy like in this case. Yet, the proxy's Status subresource might indicate
		// ongoing changes, but it's not a realistic expectation to coordinate in such cases.
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], false)                   // check lock status based on old sync group
		conduits[0].Annotations[updateSyncGroupKey] = "new-test-group"        // update the sync group in local conduits list
		assertLockStatus(t, reconciler, conduits[0], false)                   // check lock status based on new sync group
		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups())) // slow conduit is still part of a sync group
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

		// Fourth reconcile - normal conduit can execute the update
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[1], false)

		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("ParallelSlowConduitUpdate_DifferentSyncGroups_RemoveConduitAnnotation", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group-A"),
			newConduit("test-conduit-B", "test-trench", "test-group-B"),
			newConduit("test-conduit-C", "test-trench", "test-group-C"),
			newConduit("test-conduit-D", "test-trench", "test-group-D"),
		}

		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 1)) // Slow conduit
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		// Simulate parallel reconcile calls using goroutines
		var wg sync.WaitGroup
		wg.Add(len(conduits))
		for _, c := range conduits {
			go func(conduit *meridiov1.Conduit) {
				defer wg.Done()
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}}
				result, err := reconciler.Reconcile(context.Background(), request)
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
				assertLockStatus(t, reconciler, conduit, true) // Conduit should hold its lock
				_ = assertDaemonSet(t, fakeClient, conduit)
			}(c)
		}
		wg.Wait()

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, len(conduits), len(reconciler.GetUpdateLocks()))

		// Simulate parallel annotation removal
		var deleteWg sync.WaitGroup
		deleteWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer deleteWg.Done()
				c := &meridiov1.Conduit{}
				err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}, c)
				assert.NoError(t, err)
				c.Annotations = make(map[string]string) // Remove all annotations, including update sync group
				err = fakeClient.Update(context.Background(), c)
				assert.NoError(t, err)

				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: c.Name, Namespace: c.Namespace}}
				result, err := reconciler.Reconcile(context.Background(), request)
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
				assertLockStatus(t, reconciler, c, false)
			}()
		}
		deleteWg.Wait()

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("ParallelConduitUpdateAndDelete_DifferentSyncGroups", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group-A"),
			newConduit("test-conduit-B", "test-trench", "test-group-B"),
			newConduit("test-conduit-C", "test-trench", "test-group-C"),
			newConduit("test-conduit-D", "test-trench", "test-group-D"),
		}

		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 0))
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		// Simulate parallel reconcile calls using goroutines
		var wg sync.WaitGroup
		wg.Add(len(conduits))
		for _, c := range conduits {
			go func(conduit *meridiov1.Conduit) {
				defer wg.Done()
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}}
				result, err := reconciler.Reconcile(context.Background(), request)
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
				assertLockStatus(t, reconciler, conduit, false) // Conduit should not hold its lock (update is immediate)
				_ = assertDaemonSet(t, fakeClient, conduit)
			}(c)
		}
		wg.Wait()

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

		// Simulate parallel deletions
		var deleteWg sync.WaitGroup
		deleteWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer deleteWg.Done()
				c := &meridiov1.Conduit{}
				err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}, c)
				assert.NoError(t, err)
				err = fakeClient.Delete(context.Background(), c)
				assert.NoError(t, err)
				reconciler.HandleConduitDeletion(c.Name, c.Namespace)
				assertLockStatus(t, reconciler, conduit, false)
			}()
		}
		deleteWg.Wait()

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("ParallelSlowConduitUpdateAndDelete_DifferentSyncGroups_ResourcesCleaned", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", "test-group-A"),
			newConduit("test-conduit-B", "test-trench", "test-group-B"),
			newConduit("test-conduit-C", "test-trench", "test-group-C"),
			newConduit("test-conduit-D", "test-trench", "test-group-D"),
		}

		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 1)) // Slow conduit
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		// Simulate parallel reconcile calls
		var reconcileWg sync.WaitGroup
		reconcileWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer reconcileWg.Done()
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}}
				result, err := reconciler.Reconcile(context.Background(), request)
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
				assertLockStatus(t, reconciler, conduit, true) // Slow conduit should hold its lock
				_ = assertDaemonSet(t, fakeClient, conduit)
			}()
		}
		reconcileWg.Wait()

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, len(conduits), len(reconciler.GetUpdateLocks()))

		// Simulate parallel deletions
		var deleteWg sync.WaitGroup
		deleteWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer deleteWg.Done()
				c := &meridiov1.Conduit{}
				err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}, c)
				assert.NoError(t, err)
				err = fakeClient.Delete(context.Background(), c)
				assert.NoError(t, err)
				reconciler.HandleConduitDeletion(c.Name, c.Namespace)
				assertLockStatus(t, reconciler, conduit, false)
			}()
		}
		deleteWg.Wait()

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups())) // Conduits were deleted
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("ParallelSlowConduitUpdateAndDelete_NoSyncGroups", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", ""),
			newConduit("test-conduit-B", "test-trench", ""),
			newConduit("test-conduit-C", "test-trench", ""),
			newConduit("test-conduit-D", "test-trench", ""),
		}

		var objects []client.Object
		for _, c := range conduits {
			objects = append(objects, c, newProxyDaemonSet(c, 1)) // Slow conduit
		}
		reconciler, fakeClient := setupReconciler(t, objects...)

		// Simulate parallel reconcile calls
		var reconcileWg sync.WaitGroup
		reconcileWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer reconcileWg.Done()
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}}
				result, err := reconciler.Reconcile(context.Background(), request)
				assert.NoError(t, err)
				assert.Equal(t, ctrl.Result{}, result)
				assertLockStatus(t, reconciler, conduit, false) // Conduit should not hold lock as it is not part of an update sync group
				_ = assertDaemonSet(t, fakeClient, conduit)
			}()
		}
		reconcileWg.Wait()

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))

		// Simulate parallel deletions
		var deleteWg sync.WaitGroup
		deleteWg.Add(len(conduits))
		for _, c := range conduits {
			conduit := c
			go func() {
				defer deleteWg.Done()
				c := &meridiov1.Conduit{}
				err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduit.Name, Namespace: conduit.Namespace}, c)
				assert.NoError(t, err)
				err = fakeClient.Delete(context.Background(), c)
				assert.NoError(t, err)
				reconciler.HandleConduitDeletion(c.Name, c.Namespace)
				assertLockStatus(t, reconciler, conduit, false)
			}()
		}
		deleteWg.Wait()

		assert.Equal(t, 0, len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

	t.Run("UpdateConduits_NoSyncGroup_AddSharedSyncGroupAnnotation", func(t *testing.T) {

		conduits := []*meridiov1.Conduit{
			newConduit("test-conduit-A", "test-trench", ""),
			newConduit("test-conduit-B", "test-trench", ""),
		}

		var objects []client.Object
		objects = append(objects, conduits[0], newProxyDaemonSet(conduits[0], 1)) // Slow conduit
		objects = append(objects, conduits[1], newProxyDaemonSet(conduits[1], 0)) // Normal conduit
		reconciler, fakeClient := setupReconciler(t, objects...)

		// Annotate both conduits to become part of the same test-group
		for i := range conduits {
			c := &meridiov1.Conduit{}
			err := fakeClient.Get(context.Background(), types.NamespacedName{Name: conduits[i].Name, Namespace: conduits[i].Namespace}, c)
			assert.NoError(t, err)
			c.Annotations = make(map[string]string)
			c.Annotations[updateSyncGroupKey] = "test-group"
			err = fakeClient.Update(context.Background(), c)
			assert.NoError(t, err)

			conduits[i].Annotations = make(map[string]string)
			conduits[i].Annotations[updateSyncGroupKey] = "test-group"
		}

		// First conduit Reconcile should get the lock
		cs := assertDaemonSet(t, fakeClient, conduits[0])
		request := reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err := reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], true)
		ds := assertDaemonSet(t, fakeClient, conduits[0])
		assert.False(t, equality.Semantic.DeepEqual(ds.Spec, cs.Spec)) // update executed

		// Second conduit Reconcile should wait for the lock
		cs = assertDaemonSet(t, fakeClient, conduits[1])
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Greater(t, result.RequeueAfter, time.Duration(0))
		assertLockStatus(t, reconciler, conduits[1], false)
		ds = assertDaemonSet(t, fakeClient, conduits[1])
		assert.True(t, equality.Semantic.DeepEqual(ds.Spec, cs.Spec)) // update not executed

		// Simulate slow conduit reaching ready state by manually
		// updating its daemonsets DesiredNumberScheduled to 0
		ds = &appsv1.DaemonSet{}
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: common.ProxyDeploymentName(conduits[0]), Namespace: conduits[0].Namespace}, ds)
		assert.NoError(t, err)
		ds.Status.DesiredNumberScheduled = 0
		err = fakeClient.Status().Update(context.Background(), ds)
		assert.NoError(t, err)

		// First conduit's next reconcile call should be successful
		// and lock must be released
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[0].Name, Namespace: conduits[0].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[0], false)

		// Second conduit's next reconcile call should be successful
		cs = assertDaemonSet(t, fakeClient, conduits[1])
		request = reconcile.Request{NamespacedName: types.NamespacedName{Name: conduits[1].Name, Namespace: conduits[1].Namespace}}
		result, err = reconciler.Reconcile(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assertLockStatus(t, reconciler, conduits[1], false)
		ds = assertDaemonSet(t, fakeClient, conduits[1])
		assert.False(t, equality.Semantic.DeepEqual(ds.Spec, cs.Spec)) // update executed

		assert.Equal(t, len(conduits), len(reconciler.GetUpdateSyncGroups()))
		assert.Equal(t, 0, len(reconciler.GetUpdateLocks()))
	})

}

// TestProxyModelLoader loads a proxy model for testing purposes
type TestProxyModelLoader struct {
}

func (t *TestProxyModelLoader) Load() (*appsv1.DaemonSet, error) {
	return t.getTestDaemonSet(), nil
}

func (t *TestProxyModelLoader) getTestDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "proxy",
			Labels: map[string]string{
				"app": "proxy",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "proxy",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 "proxy",
						"spiffe.io/spiffe-id": "true",
					},
				},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Name:  "sysctl-init",
							Image: "registry.nordix.org/cloud-native/meridio/busybox:1.29",
							SecurityContext: &v1.SecurityContext{
								Privileged: &[]bool{true}[0],
							},
							Command: []string{"/bin/sh"},
							Args:    []string{}, // To be filled by operator according to the Trench
						},
					},
					Containers: []v1.Container{
						{
							Name:  "proxy",
							Image: "registry.nordix.org/cloud-native/meridio/proxy:latest",
							Env: []v1.EnvVar{
								{
									Name:  "SPIFFE_ENDPOINT_SOCKET",
									Value: "unix:///run/spire/sockets/agent.sock",
								},
								{
									Name: "NSM_NAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "NSM_HOST",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "spire-agent-socket",
									MountPath: "/run/spire/sockets",
									ReadOnly:  true,
								},
								{
									Name:      "nsm-socket",
									MountPath: "/var/lib/networkservicemesh",
									ReadOnly:  true,
								},
								{
									Name:      "tmp",
									MountPath: "/tmp",
									ReadOnly:  false,
								},
							},
							SecurityContext: &v1.SecurityContext{
								RunAsNonRoot:           &[]bool{true}[0],
								ReadOnlyRootFilesystem: &[]bool{true}[0],
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"all"},
									Add:  []v1.Capability{"NET_ADMIN", "DAC_OVERRIDE", "NET_RAW", "SYS_PTRACE"},
								},
							},
						},
					},
					SecurityContext: &v1.PodSecurityContext{
						FSGroup: &[]int64{2000}[0],
					},
					Volumes: []v1.Volume{
						{
							Name: "spire-agent-socket",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/run/spire/sockets",
									Type: &[]v1.HostPathType{v1.HostPathDirectory}[0],
								},
							},
						},
						{
							Name: "nsm-socket",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/networkservicemesh",
									Type: &[]v1.HostPathType{v1.HostPathDirectoryOrCreate}[0],
								},
							},
						},
						{
							Name: "tmp",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}
