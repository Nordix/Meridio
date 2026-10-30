/*
Copyright (c) 2021 Nordix Foundation

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

package sqlite

import (
	"context"
	"errors"
	"fmt"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp/registry/common"
	"github.com/nordix/meridio/pkg/nsp/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TargetRegistrySQLite struct {
	DB       *gorm.DB
	mu       sync.Mutex
	watchers map[*common.RegistryWatcher]struct{}
	// notifyDirty is true when watchers may be behind the DB: a write succeeded
	// but no broadcast has succeeded since. Set then re-notifies instead of
	// taking the fast path. Guarded by mu. False at startup is correct: watchers
	// are in-memory only, so nothing is owed across a restart.
	notifyDirty bool
}

func New(datastore string) (*TargetRegistrySQLite, error) {
	db, err := gorm.Open(sqlite.Open(datastore), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		FullSaveAssociations: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open db session: %w", err)
	}
	targetRegistrySQLite := &TargetRegistrySQLite{
		DB:       db,
		watchers: make(map[*common.RegistryWatcher]struct{}),
	}
	err = targetRegistrySQLite.init()
	if err != nil {
		return nil, err
	}
	return targetRegistrySQLite, nil
}

func (trsql *TargetRegistrySQLite) Close() error {
	sqlDB, err := trsql.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to close db connection: %w", err)
	}
	sqlDB.Close()
	return nil
}

func (trsql *TargetRegistrySQLite) Set(ctx context.Context, target *nspAPI.Target) error {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	var t Target
	targetModel := NSPTargetToSQLTarget(target)
	tx := trsql.DB.First(&t, "ID", targetModel.ID)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) { // Create
			tx = trsql.DB.Create(targetModel)
			if tx.Error != nil {
				return tx.Error
			}
			trsql.notifyDirty = true // cleared by a successful broadcast
			err := trsql.clearConflicts(ctx, target)
			if err != nil {
				return err
			}
			return trsql.notifyAllWatchers()
		}
		return tx.Error
	}
	// update
	// Fast path: targets refresh periodically (keepalive), and most refreshes
	// change nothing. If the stored row matches and watchers are not behind,
	// skip Save, clearConflicts and the broadcast, which re-reads the whole
	// target table on every refresh.
	//
	// notifyDirty is set after a write until a broadcast succeeds. While set,
	// re-broadcast instead of skipping; otherwise a failed broadcast would never
	// be retried, because every later unchanged refresh would skip.
	//
	// Skipping loses nothing: a broadcast sends the full target list, and watchers
	// already got it. A watcher whose Send failed reconnects and gets a fresh full
	// list from Watch.
	if t.Equals(targetModel) {
		if !trsql.notifyDirty {
			return nil
		}
		return trsql.notifyAllWatchers()
	}
	tx = trsql.DB.Save(&targetModel)
	if tx.Error != nil {
		return tx.Error
	}
	trsql.notifyDirty = true // cleared by a successful broadcast
	err := trsql.clearConflicts(ctx, target)
	if err != nil {
		return err
	}
	return trsql.notifyAllWatchers()
}

func (trsql *TargetRegistrySQLite) Remove(ctx context.Context, target *nspAPI.Target) error {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	targetModel := NSPTargetToSQLTarget(target)
	return trsql.remove(ctx, targetModel)
}

func (trsql *TargetRegistrySQLite) remove(ctx context.Context, target *Target) error {
	tx := trsql.DB.Delete(target)
	if tx.Error != nil {
		return tx.Error
	}
	return trsql.notifyAllWatchers()
}

func (trsql *TargetRegistrySQLite) Watch(ctx context.Context, target *nspAPI.Target) (types.TargetWatcher, error) {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	trsql.setWatchersIfNil()
	watcher := common.NewRegistryWatcher(target)
	trsql.watchers[watcher] = struct{}{}
	targets, err := trsql.getAll()
	watcher.Notify(targets)
	return &sqliteWatcher{RegistryWatcher: watcher, registry: trsql}, err
}

func (trsql *TargetRegistrySQLite) Get(ctx context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	return trsql.get(ctx, target)
}

func (trsql *TargetRegistrySQLite) notifyAllWatchers() error {
	trsql.setWatchersIfNil()
	targets, err := trsql.getAll()
	if err != nil {
		trsql.notifyDirty = true // watchers stay behind the DB
		return err
	}
	for watcher := range trsql.watchers {
		watcher.Notify(targets) // no-op if stopped; sqliteWatcher.Stop removes it
	}
	trsql.notifyDirty = false // watchers now match the DB
	return nil
}

func (trsql *TargetRegistrySQLite) setWatchersIfNil() {
	if trsql.watchers == nil {
		trsql.watchers = make(map[*common.RegistryWatcher]struct{})
	}
}

// removeWatcher removes a stopped watcher from the watcher list.
func (trsql *TargetRegistrySQLite) removeWatcher(watcher *common.RegistryWatcher) {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	delete(trsql.watchers, watcher)
}

func (trsql *TargetRegistrySQLite) get(ctx context.Context, target *nspAPI.Target) ([]*nspAPI.Target, error) {
	targets, err := trsql.getAll()
	if err != nil {
		return []*nspAPI.Target{}, err
	}
	return common.Filter(target, targets), nil
}

func (trsql *TargetRegistrySQLite) getAll() ([]*nspAPI.Target, error) {
	nspTargets := []*nspAPI.Target{}
	var targets []Target
	tx := trsql.DB.Preload("Stream.Conduit.Trench").Find(&targets)
	if tx.Error != nil {
		return nspTargets, tx.Error
	}
	for _, t := range targets {
		nspT := SQLTargetToNSPTarget(&t)
		nspTargets = append(nspTargets, nspT)
	}
	return nspTargets, nil
}

func (trsql *TargetRegistrySQLite) init() error {
	err := trsql.DB.AutoMigrate(&Target{})
	if err != nil {
		return fmt.Errorf("failed to AutoMigrate target: %w", err)
	}
	err = trsql.DB.AutoMigrate(&Stream{})
	if err != nil {
		return fmt.Errorf("failed to AutoMigrate stream: %w", err)
	}
	err = trsql.DB.AutoMigrate(&Conduit{})
	if err != nil {
		return fmt.Errorf("failed to AutoMigrate conduit: %w", err)
	}
	err = trsql.DB.AutoMigrate(&Trench{})
	if err != nil {
		return fmt.Errorf("failed to AutoMigrate trench: %w", err)
	}
	return nil
}

// Due the way the data are stored, 2 objects (Stream, Conduit, Trench) with the same name but with different parent
// causes problems.
// FullSaveAssociations will update the association when a target will be registered/updated. This will cause
// old targets, with, for instance, the same stream name but different conduit name to get their conduit name
// replaced. This function will then remove all targets which got their association replaced.
//
// TODO: The exact role and necessity of clearConflicts should be clarified. It
// reconciles a shortcut in the schema: Stream/Conduit/Trench are keyed by bare
// name (namespace is part of neither the keys nor GetTargetID), so a stream name
// mapped to a different conduit collides onto the same row, whose association is
// repointed on write (FullSaveAssociations); this cleanup then deletes the
// targets left mismatched.
//
// A persistent same-name/different-conduit collision cannot be resolved: two
// such targets permanently displace each other. Each refresh re-Saves one target,
// repoints the shared association and evicts the other; the evicted one recreates
// on its next refresh and evicts the first in turn. So at most one of them is ever
// present at a time and both churn (evict + recreate, each fanned out to watchers)
// on every refresh cycle, indefinitely.
//
// Currently reachable trigger: a Stream's spec.conduit is NOT immutable (the
// Stream webhook only forbids changing the trench label and max-targets), so
// changing a Stream's conduit can trigger this path. For a genuine move (old
// conduit abandoned) it is a convergent transient: old-conduit targets are
// evicted, new-conduit ones registered, and it settles once no old-conduit target
// remains. It only becomes the unresolvable battle above if targets for both the
// old and new conduit under the same stream name keep refreshing concurrently.
// In practice this is bounded: a TAPA closes (unregisters) a stream that
// disappears from its conduit's configuration, and an unrefreshed target
// expires after NSP_ENTRY_TIMEOUT.
//
// Cross-namespace same-name streams cannot currently collide here, but only
// because the trench controller lists CRDs with InNamespace(trench.Namespace)
// when rendering the ConfigMap NSP reads, so a given NSP only ever sees one
// namespace's streams. If that scoping changed (controller aggregating multiple
// namespaces into one NSP's config), same-name streams from different namespaces
// would collide and ping-pong the same way. Namespace-aware keys (composite
// (namespace, name), namespace in GetTargetID) would remove that fragility.
func (trsql *TargetRegistrySQLite) clearConflicts(ctx context.Context, target *nspAPI.Target) error {
	if target == nil {
		return nil
	}
	var targets []Target
	streamName := ""
	if target.Stream != nil {
		streamName = target.Stream.Name
	}
	tx := trsql.DB.Preload("Stream.Conduit.Trench").Where("stream_name = ?", streamName).Find(&targets)
	if tx.Error != nil {
		return tx.Error
	}
	for _, tt := range targets {
		nspT := SQLTargetToNSPTarget(&tt)
		if tt.ID != GetTargetID(nspT) {
			err := trsql.remove(ctx, &tt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
