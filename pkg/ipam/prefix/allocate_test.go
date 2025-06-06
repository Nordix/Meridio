/*
Copyright (c) 2021 Nordix Foundation
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

package prefix_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/memory"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
)

func Test_Prefix_IPv4_Allocate(t *testing.T) {
	p := prefix.New("parent", "169.16.0.0/16", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 24, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "169.16.0.0/24", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 24, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "169.16.1.0/24", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-c", 32, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-c", newPrefix.GetName())
	assert.Equal(t, "169.16.2.0/32", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-d", 32, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-d", newPrefix.GetName())
	assert.Equal(t, "169.16.2.1/32", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-e", 17, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-e", newPrefix.GetName())
	assert.Equal(t, "169.16.128.0/17", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
}

func Test_Prefix_IPv4_Allocate_Full(t *testing.T) {
	p := prefix.New("parent", "169.16.0.0/24", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 25, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "169.16.0.0/25", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 25, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "169.16.0.128/25", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	_, err = prefix.Allocate(context.TODO(), p, "child-b", 32, store)
	assert.NotNil(t, err)
}

func Test_Prefix_IPv6_Allocate(t *testing.T) {
	p := prefix.New("parent", "2001:1::/32", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 64, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "2001:1::/64", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 64, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:1::/64", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-c", 128, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-c", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:2::/128", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-d", 128, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-d", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:2::1/128", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-e", 33, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-e", newPrefix.GetName())
	assert.Equal(t, "2001:1:8000::/33", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
}

func Test_Prefix_IPv6_Allocate_Full(t *testing.T) {
	p := prefix.New("parent", "2001:1::/64", nil)
	assert.NotNil(t, p)
	store := memory.New()
	assert.NotNil(t, store)
	newPrefix, err := prefix.Allocate(context.TODO(), p, "child-a", 65, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-a", newPrefix.GetName())
	assert.Equal(t, "2001:1::/65", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	newPrefix, err = prefix.Allocate(context.TODO(), p, "child-b", 65, store)
	assert.Nil(t, err)
	assert.Equal(t, "child-b", newPrefix.GetName())
	assert.Equal(t, "2001:1:0:0:8000::/65", newPrefix.GetCidr())
	assert.True(t, p.Equals(newPrefix.GetParent()))
	_, err = prefix.Allocate(context.TODO(), p, "child-c", 128, store)
	assert.NotNil(t, err)
}

func Test_Prefix_Allocate_Concurrency(t *testing.T) {
	t.Run("IPv4_Memory_Storage_Collision", func(t *testing.T) {
		parent, store := setupTestEnv(t, "169.16.0.0/24", setupDelayedMemoryStore)

		// phase1: try to allocate /25 prefixes in 169.16.0.0/24 as parent
		phase1Children := []string{"child-a", "child-b"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 25)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// No successful allocation is expected. Due to the in-memory store's allowance
		// of duplicate CIDR adds and the allocate logic's simple collision resolution
		// (move to next candidate without retry), concurrent attempts lead to constant
		// add/delete churn without stable allocation.
		phase1ExpectedCIDRs := map[string]struct{}{}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv4 Phase-1")

		executeOnConcreteType(
			t,
			context.TODO(),
			store,
			(*delayedStore)(nil),
			func(t *testing.T, ctx context.Context, concreteDS *delayedStore) {
				assertNoChildren(t, ctx, concreteDS, parent)
			},
		)
	})

	t.Run("IPv6_Memory_Storage_Collision", func(t *testing.T) {
		parent, store := setupTestEnv(t, "2001:1::/64", setupDelayedMemoryStore)

		// phase1: try to allocate /66 prefixes in 2001:1::/64 as parent
		phase1Children := []string{"child-a", "child-b"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 66)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// No successful allocation is expected. Due to the in-memory store's allowance
		// of duplicate CIDR adds and the allocate logic's simple collision resolution
		// (move to next candidate without retry), concurrent attempts lead to constant
		// add/delete churn without stable allocation.
		phase1ExpectedCIDRs := map[string]struct{}{}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv6 Phase-1")

		executeOnConcreteType(
			t,
			context.TODO(),
			store,
			(*delayedStore)(nil),
			func(t *testing.T, ctx context.Context, concreteDS *delayedStore) {
				assertNoChildren(t, ctx, concreteDS, parent)
			},
		)
	})

	t.Run("IPv4_Sqlite_Storage", func(t *testing.T) {
		parent, store := setupTestEnv(t, "169.16.0.0/16", setupDelayedSqliteStore)

		// Phase1: allocate /24 prefixes in 169.16.0.0/16 as parent
		phase1Children := []string{"child-a", "child-b", "child-c", "child-d"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 24)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase1ExpectedCIDRs := map[string]struct{}{
			"169.16.0.0/24": {},
			"169.16.1.0/24": {},
			"169.16.2.0/24": {},
			"169.16.3.0/24": {},
		}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv4 Phase-1")

		// Phase2: allocate /32 prefixes in previously allocated children prefixes ("169.16.x.0/24") as parents
		phase2Tasks := []allocationTask{}
		for _, p := range phase1Allocated {
			childrenForParent := []string{p.GetName() + "-1", p.GetName() + "-2", p.GetName() + "-3"}
			phase2Tasks = append(phase2Tasks, prepareAllocationTasks(p, childrenForParent, 32)...)
		}
		phase2Allocated := executeAndCollectTasks(t, context.TODO(), phase2Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase2ExpectedCIDRs := map[string]struct{}{
			"169.16.0.0/32": {},
			"169.16.0.1/32": {},
			"169.16.0.2/32": {},
			"169.16.1.0/32": {},
			"169.16.1.1/32": {},
			"169.16.1.2/32": {},
			"169.16.2.0/32": {},
			"169.16.2.1/32": {},
			"169.16.2.2/32": {},
			"169.16.3.0/32": {},
			"169.16.3.1/32": {},
			"169.16.3.2/32": {},
		}
		verifyAllocated(t, phase2Allocated, phase2ExpectedCIDRs, "IPv4 Phase-2")
	})

	t.Run("IPv6_Sqlite_Storage", func(t *testing.T) {
		parent, store := setupTestEnv(t, "2001:1::/32", setupDelayedSqliteStore)

		// Phase1: allocate /64 prefixes in 2001:1::/32 as parent
		phase1Children := []string{"child-a", "child-b", "child-c"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 64)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase1ExpectedCIDRs := map[string]struct{}{
			"2001:1::/64":     {},
			"2001:1:0:1::/64": {},
			"2001:1:0:2::/64": {},
		}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv6 Phase-1")

		// Phase2: allocate /128 prefixes in previously allocated children prefixes as parents
		phase2Tasks := []allocationTask{}
		for _, p := range phase1Allocated {
			childrenForParent := []string{p.GetName() + "-1", p.GetName() + "-2", p.GetName() + "-3"}
			phase2Tasks = append(phase2Tasks, prepareAllocationTasks(p, childrenForParent, 128)...)
		}
		phase2Allocated := executeAndCollectTasks(t, context.TODO(), phase2Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase2ExpectedCIDRs := map[string]struct{}{
			"2001:1::/128":      {},
			"2001:1::1/128":     {},
			"2001:1::2/128":     {},
			"2001:1:0:1::/128":  {},
			"2001:1:0:1::1/128": {},
			"2001:1:0:1::2/128": {},
			"2001:1:0:2::/128":  {},
			"2001:1:0:2::1/128": {},
			"2001:1:0:2::2/128": {},
		}
		verifyAllocated(t, phase2Allocated, phase2ExpectedCIDRs, "IPv6 Phase-2")
	})

	t.Run("IPv4_Sqlite_Storage_Full", func(t *testing.T) {
		parent, store := setupTestEnv(t, "169.16.0.0/24", setupDelayedSqliteStore)

		// phase1: allocate /25 prefixes in 169.16.0.0/24 as parent
		phase1Children := []string{"child-a", "child-b", "child-c", "child-d"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 25)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase1ExpectedCIDRs := map[string]struct{}{
			"169.16.0.0/25":   {},
			"169.16.0.128/25": {},
		}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv4 Phase-1")
	})

	t.Run("IPv6_Sqlite_Storage_Full", func(t *testing.T) {
		parent, store := setupTestEnv(t, "2001:1::/64", setupDelayedSqliteStore)

		// phase1: allocate /66 prefixes in 2001:1::/64 as parent
		phase1Children := []string{"child-a", "child-b", "child-c", "child-d", "child-e", "child-f"}
		phase1Tasks := prepareAllocationTasks(parent, phase1Children, 66)
		phase1Allocated := executeAndCollectTasks(t, context.TODO(), phase1Tasks, store)

		// Given the parent prefix and requested bit size it's expected that allocation picks the first available prefix
		phase1ExpectedCIDRs := map[string]struct{}{
			"2001:1::/66":          {},
			"2001:1:0:0:4000::/66": {},
			"2001:1:0:0:8000::/66": {},
			"2001:1:0:0:c000::/66": {},
		}
		verifyAllocated(t, phase1Allocated, phase1ExpectedCIDRs, "IPv6 Phase-1")
	})
}

const dbFileName = "test.db"
const storeDelay time.Duration = 20 * time.Millisecond

type delayedStoreFunc func(t *testing.T) types.Storage

// delayedStore wraps a types.Storage to introduce artificial delays
// in specific operations, enabling more effective concurrency testing.
//
// By delaying Add(), this store creates scenarios where multiple concurrent goroutines
// can attempt to allocate the same or conflicting prefixes simultaneously.
// This is crucial for verifying the robustness of prefix.Allocate(), as it exposes longer
// and more complex collision detection and resolution sequences.
//
// Additionally, delaying GetChilds() and Delete() is important because prefix.Allocate()
// typically performs an Add(), then validates with GetChilds() to check for conflicts
// (and may use Delete() on conflict). Introducing a delay here increases the likelihood
// of extended collision sequences.
type delayedStore struct {
	types.Storage
	delay time.Duration
}

func (ds *delayedStore) sleep(ctx context.Context) error {
	t := time.NewTimer(ds.delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	return nil
}

func (ds *delayedStore) Add(ctx context.Context, prefix types.Prefix) error {
	//logger := log.FromContextOrGlobal(ctx).WithName("delayedStore.Add").WithValues("Name", prefix.GetName(), "Cidr", prefix.GetCidr())
	//logger.Info("Add is delayed")

	if err := ds.sleep(ctx); err != nil {
		return err
	}
	//logger.Info("Add delay done")

	if err := ds.Storage.Add(ctx, prefix); err != nil {
		//logger.Error(err, "Add failed")
		return err
	}

	return nil
}

func (ds *delayedStore) Delete(ctx context.Context, prefix types.Prefix) error {
	//logger := log.FromContextOrGlobal(ctx).WithName("delayedStore").WithValues("Name", prefix.GetName(), "Cidr", prefix.GetCidr())
	//logger.Info("Delete is delayed")

	if err := ds.sleep(ctx); err != nil {
		return err
	}
	//logger.Info("Delete delay done")

	if err := ds.Storage.Delete(ctx, prefix); err != nil {
		//logger.Error(err, "Delete failed")
		return err
	}

	return nil
}

func (ds *delayedStore) GetChilds(ctx context.Context, prefix types.Prefix) ([]types.Prefix, error) {
	//logger := log.FromContextOrGlobal(ctx).WithName("delayedStore").WithValues("prefix", prefix)
	//logger.Info("GetChilds is delayed")

	if err := ds.sleep(ctx); err != nil {
		return nil, err
	}
	//logger.Info("GetChilds delay done")

	return ds.Storage.GetChilds(ctx, prefix)
}

func setupSqliteStore(t *testing.T) types.Storage {
	_ = os.Remove(dbFileName)
	t.Cleanup(func() {
		os.Remove(dbFileName)
	})

	sqliteStore, err := sqlite.New(dbFileName)
	assert.Nil(t, err, "Failed to create SQLite store")

	return sqliteStore
}

func setupDelayedSqliteStore(t *testing.T) types.Storage {
	sqliteStore := setupSqliteStore(t)
	assert.NotNil(t, sqliteStore, "Sqlite store should not be nil")

	return &delayedStore{
		Storage: sqliteStore,
		delay:   storeDelay,
	}
}

func setupDelayedMemoryStore(t *testing.T) types.Storage {
	memStore := memory.New()
	assert.NotNil(t, memStore, "Memory store should not be nil")

	return &delayedStore{
		Storage: memStore,
		delay:   storeDelay,
	}
}

/* // setupSqliteStoreWithNoUniquenessConstraint sets up an sqlite database that
// relies on the legacy model not enforcing CIDR uniqueness
func setupSqliteStoreWithNoUniquenessConstraint(t *testing.T) types.Storage {
	_ = os.Remove(dbFileName)
	t.Cleanup(func() {
		os.Remove(dbFileName)
	})

	type Prefix struct {
		Id        string `gorm:"primaryKey"`
		Name      string `gorm:"index"` // supposedly indexing could improve query performance
		Cidr      string
		ParentID  string `gorm:"index"`
		Parent    *Prefix
		UpdatedAt time.Time `gorm:"index"`               // supposedly indexing could improve query performance
		Expirable *bool     `gorm:"index;default:false"` // indicates whether prefix can expire and thus be subject to garbage collection
	}

	sqliteStore, err := func(datastore string) (types.Storage, error) {
		db, err := gorm.Open(sqliteDrv.Open(datastore), &gorm.Config{
			Logger:  logger.Default.LogMode(logger.Silent),
			NowFunc: func() time.Time { return time.Now().UTC() },
		})
		if err != nil {
			return nil, fmt.Errorf("failed to open db session: %w", err)
		}
		sqlis := &sqlite.SQLiteIPAMStorage{
			DB: db,
		}

		err = sqlis.DB.AutoMigrate(Prefix{})
		if err != nil {
			return nil, fmt.Errorf("failed to automigrate: %w", err)
		}

		return sqlis, nil
	}(dbFileName)

	assert.Nil(t, err, "Failed to create SQLite store")

	return sqliteStore
}

func setupDelayedSqliteStoreWithNoUniquenessConstraint(t *testing.T) types.Storage {
	sqliteStore := setupSqliteStoreWithNoUniquenessConstraint(t)
	assert.NotNil(t, sqliteStore, "Sqlite store should not be nil")

	return &delayedStore{
		Storage: sqliteStore,
		delay:   storeDelay,
	}
} */

// setupTestEnv sets up the parent prefix, and a delayed store
func setupTestEnv(t *testing.T, parentCIDR string, setupDelayedStore delayedStoreFunc) (types.Prefix, types.Storage) {
	assert.NotNil(t, setupDelayedStore, "Delayed store setup function should not be nil")
	store := setupDelayedStore(t)
	assert.NotNil(t, store, "DelayedAddStore should not be nil")

	parent := prefix.New("parent", parentCIDR, nil)
	assert.NotNil(t, parent, "Parent prefix should not be nil")

	return parent, store
}

// allocationTask encapsulates arguments for a single prefix.Allocate() call
type allocationTask struct {
	Parent    types.Prefix
	ChildName string
	BitSize   int
}

// prepareAllocationTasks creates allocation tasks for given parent, children and bitSize
func prepareAllocationTasks(parent types.Prefix, childNames []string, bitSize int) []allocationTask {
	tasks := make([]allocationTask, 0, len(childNames))
	for _, childName := range childNames {
		tasks = append(tasks, allocationTask{Parent: parent, ChildName: childName, BitSize: bitSize})
	}
	return tasks
}

// runConcurrentAllocations launches parallel worker goroutines to perform allocation tasks
// Workers send successfully allocated prefixes into allocatedCh.
// Note: Failed allocations are logged but do not abort test execution as certain test cases
// might expect failed attempts.
func runConcurrentAllocations(
	t *testing.T,
	ctx context.Context,
	wg *sync.WaitGroup,
	tasks []allocationTask,
	store types.Storage,
	allocatedCh chan<- types.Prefix,
) {
	wg.Add(len(tasks))

	for _, task := range tasks {
		go func(task *allocationTask) {
			defer wg.Done()

			newPrefix, err := prefix.Allocate(ctx, task.Parent, task.ChildName, task.BitSize, store)
			if err != nil {
				t.Logf("Failed to allocate prefix for child %q, err: %v", task.ChildName, err)
			} else {
				assert.Equal(t, task.ChildName, newPrefix.GetName())
				assert.True(t, task.Parent.Equals(newPrefix.GetParent()))

				t.Logf("Allocated prefix for child %q: %s", task.ChildName, newPrefix.GetCidr())
				allocatedCh <- newPrefix // let caller collect allocated prefixes
			}
		}(&task)
	}
}

// executeAndCollectTasks collects results from runConcurrentAllocations() and waits
// for all the workers to finish
// Returns a map of CIDR to Prefix for successfully allocated prefixes.
// Expects no duplicate CIDR allocation.
func executeAndCollectTasks(
	t *testing.T,
	ctx context.Context,
	tasks []allocationTask,
	store types.Storage,
) map[string]types.Prefix {
	result := make(chan types.Prefix)
	allocateMap := make(map[string]types.Prefix) // maps CIDR to Prefix

	var wg sync.WaitGroup
	runConcurrentAllocations(t, ctx, &wg, tasks, store, result)

	// Wait for all workers then close the results channel
	go func() {
		wg.Wait()
		close(result)
	}()

	// Collect results from the channel into the map
	for p := range result {
		_, ok := allocateMap[p.GetCidr()] // assert no duplicate CIDRs are allocated
		assert.False(t, ok, "Duplicate CIDR allocated: %s (name: %s)", p.GetCidr(), p.GetName())
		allocateMap[p.GetCidr()] = p
	}

	return allocateMap
}

// verifyAllocated checks if the actual allocated CIDRs match the expected ones
func verifyAllocated(t *testing.T, actual map[string]types.Prefix, expected map[string]struct{}, phase string) {
	assert.Equal(t, len(expected), len(actual), "Mismatch in number of allocated prefixes for %s", phase)

	actualCIDRs := []string{}
	for _, p := range actual {
		_, ok := expected[p.GetCidr()]
		assert.True(t, ok, "Unexpected CIDR allocated in %s: %s (name: %s)", phase, p.GetCidr(), p.GetName())
		delete(expected, p.GetCidr()) // remove found CIDR from expected set
		actualCIDRs = append(actualCIDRs, p.GetCidr())
	}
	// Expected map should be empty if all were found
	assert.Len(t, expected, 0, "Not all expected CIDRs were allocated in %s (actual: %v)", phase, actualCIDRs)
}

func executeOnConcreteType[T any](
	t *testing.T,
	ctx context.Context,
	subject any,
	expectedConcreteType T, // a nil value of the concrete type T to infer T (e.g., (*delayedStore)(nil))
	testLogic func(t *testing.T, ctx context.Context, concreteT T),
) {
	concreteValue, ok := subject.(T)
	assert.True(t, ok, "Failed to assert subject to expected type %T. Actual type: %T", expectedConcreteType, subject)

	if !ok {
		return
	}

	if testLogic != nil {
		testLogic(t, ctx, concreteValue)
	}
}

// assertNoChildren checks that the wrapped storage contains no children
// for the given parent prefix
func assertNoChildren(t *testing.T, ctx context.Context, ds *delayedStore, parent types.Prefix) {
	children, err := ds.Storage.GetChilds(ctx, parent) // calling GetChilds on the "wrapped" store to avoid delayed execution
	assert.Nil(t, err)
	assert.Len(t, children, 0, "Parent prefix should not contain any children")
}
