/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024 OpenInfra Foundation Europe

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
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/nordix/meridio/pkg/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const srcChildNamePrefix = "-src"
const dstChildNamePrefix = "-dst"

type SQLiteIPAMStorage struct {
	DB *gorm.DB
	mu sync.Mutex
}

func New(datastore string) (*SQLiteIPAMStorage, error) {
	db, err := gorm.Open(sqlite.Open(datastore), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open db session: %w", err)
	}
	sqlis := &SQLiteIPAMStorage{
		DB: db,
	}
	err = sqlis.init()
	if err != nil {
		return nil, err
	}
	return sqlis, nil
}

// StartGarbageCollector -
// StartGarbageCollector starts a one time garbage collector to remove
// leftover data created based on the old model. Also, periodically runs
// a garbage collector to clean up outdated records with valid updatedAt
// values (where threshold determines what is considered outdated).
func (sqlis *SQLiteIPAMStorage) StartGarbageCollector(ctx context.Context, interval time.Duration, threshold time.Duration) {
	var once sync.Once
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		/*
			initialDelay := 1 * time.Minute
			timer := time.NewTimer(initialDelay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				select {
				case <-timer.C:
				default:
				}
				return
			} */

		// Run the periodic task
		for {
			select {
			case <-ctx.Done():
				return // Exit the goroutine when the stop signal is received
			case <-ticker.C:
				once.Do(func() {
					if err := sqlis.oneTimeGarbageCollector(ctx); err != nil {
						log.Logger.Info("One time garbage collector returned error", "err", err)
					}
				})
				if err := sqlis.garbageCollector(ctx, threshold); err != nil {
					log.Logger.Info("Garbage collector returned error", "err", err)
				}
			}
		}
	}()
}

// Fetch and delete -src/-dst records whose updatedAt timestamp is considered
// expired according to the threshold.
func (sqlis *SQLiteIPAMStorage) garbageCollector(ctx context.Context, threshold time.Duration) error {
	// Define the batch size
	batchSize := 50
	likeDst := "%" + dstChildNamePrefix
	likeSrc := "%" + srcChildNamePrefix
	logger := log.Logger.WithValues("func", "garbageCollector")
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Start a transaction for batch deletion
			// Note: It's not really expected to have a huge table, yet batch
			// processing approach was chosen.
			// Note: Preferred printing entries to be deleted, hence the two
			// separate actions to fetch and then delete entries.
			tx := sqlis.DB.Begin()
			if tx.Error != nil {
				return tx.Error
			}
			entryThreshHold := time.Now().UTC().Add(-threshold)

			// Fetch entries to be deleted
			var deleteEntries []Prefix
			if err := tx.Where("(updated_at IS NOT NULL AND updated_at != ? AND updated_at < ?) AND (name LIKE ? OR name LIKE ?)",
				time.Time{}, entryThreshHold, likeSrc, likeDst).Limit(batchSize).Find(&deleteEntries).Error; err != nil {
				tx.Rollback()
				return err
			}

			// No more entries to delete
			if len(deleteEntries) == 0 {
				tx.Rollback()
				return nil
			}

			// Log the fetched entries
			for _, entry := range deleteEntries {
				logger.V(1).Info("to delete", "entry", entry)
			}

			// Delete the fetched entries
			idsToDelete := make([]string, len(deleteEntries))
			for i, entry := range deleteEntries {
				idsToDelete[i] = entry.Id
			}

			if err := tx.Where("id IN ?", idsToDelete).Delete(&Prefix{}).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Info("entry to delete not found", "err", err)
				} else {
					tx.Rollback()
					return err
				}
			}

			// Commit the transaction
			if err := tx.Commit().Error; err != nil {
				return err
			}

			logger.Info("deleted entries in transaction", "count", len(idsToDelete))
		}
	}
}

// oneTimeGarbageCollector -
// Removes records created based on the old model to clean up possible leaks.
// Matches records whose name has "-src"/"-dst" suffix. Must be invoked with
// a carefully chosen delay to avoid removing entries still in use (to allow
// upate of such records when just upgraded from an old model).
//
// Note: When called removes all matching entries!!!
func (sqlis *SQLiteIPAMStorage) oneTimeGarbageCollector(ctx context.Context) error {
	// Define the batch size
	logger := log.Logger.WithValues("func", "oneTimeGarbageCollector")
	batchSize := 50
	likeDst := "%" + dstChildNamePrefix
	likeSrc := "%" + srcChildNamePrefix
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Start a transaction for batch deletion
			// Note: It's not really expected to have a huge table, yet batch
			// processing approach was chosen.
			// Note: Preferred printing entries to be deleted, hence the two
			// separate actions to fetch and then delete entries.
			tx := sqlis.DB.Begin()
			if tx.Error != nil {
				return tx.Error
			}

			// Fetch entries created according to the old model which haven't
			// been updated since the migration and the initial delay.
			var deleteEntries []Prefix
			if err := tx.Where("(updated_at IS NULL OR updated_at = ?) AND (name LIKE ? OR name LIKE ?)",
				time.Time{}, likeSrc, likeDst).Limit(batchSize).Find(&deleteEntries).Error; err != nil {
				tx.Rollback()
				return err
			}

			// No more entries to delete
			if len(deleteEntries) == 0 {
				tx.Rollback()
				return nil
			}

			// Log the fetched entries
			for _, entry := range deleteEntries {
				logger.V(1).Info("to delete", "entry", entry)
			}

			// Delete the fetched entries
			idsToDelete := make([]string, len(deleteEntries))
			for i, entry := range deleteEntries {
				idsToDelete[i] = entry.Id
			}

			if err := tx.Where("id IN ?", idsToDelete).Delete(&Prefix{}).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					logger.Info("entry to delete not found", "err", err)
				} else {
					tx.Rollback()
					return err
				}
			}

			// Commit the transaction
			if err := tx.Commit().Error; err != nil {
				return err
			}

			logger.Info("deleted entries in transaction", "count", len(idsToDelete))
		}
	}
}

func (sqlis *SQLiteIPAMStorage) Add(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	model := prefixToModel(prefix)
	if model == nil {
		return nil
	}
	tx := sqlis.DB.Create(model)
	return tx.Error
}

// Update -
// Updates the database entry if present.
// Currently, the whole purpose of this function is to update the UpdatedAt
// field in the database that is used by garbage collector logic to clean up
// unused entries.
func (sqlis *SQLiteIPAMStorage) Update(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	model := prefixToModel(prefix)
	if model == nil {
		return nil
	}
	tx := sqlis.DB.Save(model)
	return tx.Error
}

func (sqlis *SQLiteIPAMStorage) Delete(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	if prefix == nil {
		return nil
	}
	return sqlis.delete(prefix)
}

func (sqlis *SQLiteIPAMStorage) Get(ctx context.Context, name string, parent types.Prefix) (types.Prefix, error) {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	prefix := prefix.New(name, "", parent)
	model := prefixToModel(prefix)
	var result *Prefix
	tx := sqlis.DB.First(&result, model)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, tx.Error
	}
	np := modelToPrefix(result, parent)
	return np, nil
}

func (sqlis *SQLiteIPAMStorage) GetChilds(ctx context.Context, prefix types.Prefix) ([]types.Prefix, error) {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	return sqlis.getChilds(prefix)
}

func (sqlis *SQLiteIPAMStorage) init() error {
	// Note: When a new column is added to an existing table using AutoMigrate,
	// then the DB initializes this column to 'NULL' for all existing entries.
	// An updated query would be required if we wanted to set a different value
	// for existing entries. Currently, the default behaviour suits our needs.
	err := sqlis.DB.AutoMigrate(&Prefix{})
	if err != nil {
		return fmt.Errorf("failed to automigrate: %w", err)
	}
	return nil
}

func (sqlis *SQLiteIPAMStorage) getChilds(prefix types.Prefix) ([]types.Prefix, error) {
	model := prefixToModel(prefix)
	var results []*Prefix
	if model == nil {
		return []types.Prefix{}, nil
	}
	tx := sqlis.DB.Where("parent_id = ?", model.Id).Find(&results)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return []types.Prefix{}, nil
		}
		return nil, tx.Error
	}
	prefixList := []types.Prefix{}
	for _, result := range results {
		np := modelToPrefix(result, prefix)
		prefixList = append(prefixList, np)
	}
	return prefixList, nil
}

func (sqlis *SQLiteIPAMStorage) delete(prefix types.Prefix) error {
	childs, err := sqlis.getChilds(prefix)
	if err != nil {
		return err
	}
	for _, child := range childs {
		err = sqlis.delete(child)
		if err != nil {
			return err
		}
	}
	model := prefixToModel(prefix)
	tx := sqlis.DB.Delete(model)
	return tx.Error
}
