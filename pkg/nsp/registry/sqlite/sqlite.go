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
}

func New(datastore string) (types.TargetRegistry, error) {
	db, err := gorm.Open(sqlite.Open(datastore), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
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
			return trsql.notifyAllWatchers()
		}
		return tx.Error
	}
	// update
	tx = trsql.DB.Save(&targetModel)
	if tx.Error != nil {
		return tx.Error
	}
	return trsql.notifyAllWatchers()
}

func (trsql *TargetRegistrySQLite) Remove(ctx context.Context, target *nspAPI.Target) error {
	trsql.mu.Lock()
	defer trsql.mu.Unlock()
	targetModel := NSPTargetToSQLTarget(target)
	tx := trsql.DB.Delete(targetModel)
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
	return watcher, err
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
		return err
	}
	for watcher := range trsql.watchers {
		if watcher.IsStopped() {
			delete(trsql.watchers, watcher)
		}
		watcher.Notify(targets)
	}
	return nil
}

func (trsql *TargetRegistrySQLite) setWatchersIfNil() {
	if trsql.watchers == nil {
		trsql.watchers = make(map[*common.RegistryWatcher]struct{})
	}
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
		return err
	}
	err = trsql.DB.AutoMigrate(&Stream{})
	if err != nil {
		return err
	}
	err = trsql.DB.AutoMigrate(&Conduit{})
	if err != nil {
		return err
	}
	err = trsql.DB.AutoMigrate(&Trench{})
	return err
}
