/*
Copyright (c) 2021 Nordix Foundation
Copyright (c) 2024-2025 OpenInfra Foundation Europe

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

const bridgeName = "bridge" // prefix with name "bridge" is special because it's not periodically updated

var ErrCIDRConflict = prefix.ErrCIDRConflict // alias to simplify usage in this package

type SQLiteIPAMStorage struct {
	DB              *gorm.DB
	mu              sync.Mutex
	prefixTableName string
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
	sqlis.prefixTableName = TableNameForModel(sqlis.DB, &Prefix{}) // Infer table name
	log.Logger.Info("Initialized prefix table name", "table", sqlis.prefixTableName)
	return sqlis, nil
}

// StartGarbageCollector -
// StartGarbageCollector periodically runs a garbage collector to clean up
// outdated expirable records with valid updatedAt values (where threshold
// determines what is considered outdated).
func (sqlis *SQLiteIPAMStorage) StartGarbageCollector(ctx context.Context, interval time.Duration, threshold time.Duration) {
	go func() {
		log.Logger.Info("Start Garbage Collector")
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
				if err := sqlis.garbageCollector(ctx, threshold); err != nil {
					log.Logger.Info("Garbage collector returned error", "err", err)
				}
			}
		}
	}()
}

// Fetch and recursively delete expirable prefixes (and their descendants)
// whose updatedAt timestamp is considered expired according to the threshold
// (i.e. was not updated for a "long time").
// Note: Even though records are processed in batches, no need to use offsets
// or pagination, because matching records are to be deleted. Thus, eventually
// there will be no more database records matching the query criteria (no risk
// of infinite loop).
func (sqlis *SQLiteIPAMStorage) garbageCollector(ctx context.Context, threshold time.Duration) error {
	// Define the batch size
	batchSize := 50
	logger := log.Logger.WithValues("func", "garbageCollector")
	sqlis.mu.Lock() // TODO: Consider moving locking within the for loop
	defer sqlis.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Start a transaction for batch deletion
			// Note: It's not really expected to have a huge table, yet batch
			// processing approach was chosen.
			// Note: Preferred printing entries eligible for removal, hence
			// the two separate actions to fetch and then delete entries.
			tx := sqlis.DB.Begin()
			if tx.Error != nil {
				return tx.Error
			}
			entryThreshHold := time.Now().UTC().Add(-threshold)

			// Fetch entries eligible for removal
			var batchOfEligiblePrefixes []Prefix
			if err := tx.Where("expirable = true AND (updated_at IS NOT NULL AND updated_at != ? AND updated_at < ?) AND name != ?",
				time.Time{}, entryThreshHold, bridgeName).Limit(batchSize).Find(&batchOfEligiblePrefixes).Error; err != nil {
				tx.Rollback()
				return err
			}

			// No more entries eligible for removal
			if len(batchOfEligiblePrefixes) == 0 {
				tx.Rollback()
				return nil
			}

			// Extract IDs of the entries found in this batch.
			idsToAnchorCTERecursion := make([]string, len(batchOfEligiblePrefixes))
			for i, entry := range batchOfEligiblePrefixes {
				logger.V(1).Info("Prefix identified for GC recursive deletion", "prefix", entry)
				idsToAnchorCTERecursion[i] = entry.Id
			}

			// Construct the SQL WITH RECURSIVE Common Table Expression (CTE) for efficient
			// hierarchical deletion. Pass extracted IDs of eligible entries as anchors for
			// the CTE in order to identify descendants down the hierarchy well.
			// (Opted for CTE instead of pure go code based recursive deletion that can have
			// poor performance for bulk operations due to numerous individual db queries.
			// Also, did not want to introduce database-level foreign key constraints for
			// the Delete method to automatically cascade.)
			sqlQuery := fmt.Sprintf(`
            WITH RECURSIVE items_to_delete AS (
                SELECT id
                FROM "%s"
                WHERE id IN ?

                UNION ALL

                SELECT child.id
                FROM "%s" AS child
                JOIN items_to_delete AS parent ON child.parent_id = parent.id
            )
            DELETE FROM "%s"
            WHERE id IN (SELECT id FROM items_to_delete);
            `, sqlis.prefixTableName, sqlis.prefixTableName, sqlis.prefixTableName)

			// Execute the recursive DELETE statement within the current transaction
			execResult := tx.WithContext(ctx).Exec(sqlQuery, idsToAnchorCTERecursion)
			if execResult.Error != nil {
				tx.Rollback()
				return fmt.Errorf("failed to perform recursive batch delete in GC using CTE: %w", execResult.Error)
			}

			// Commit the transaction
			if err := tx.Commit().Error; err != nil {
				return err
			}

			logger.Info("Recursively deleted prefixes in transaction using CTE",
				"total_rows_affected", execResult.RowsAffected, "table", sqlis.prefixTableName)

			// No need for a new query if we got less items than the batchSize this round
			if len(batchOfEligiblePrefixes) < batchSize {
				return nil
			}
		}
	}
}

// Add adds prefix to database.
// Also sets the expirable field based on the context.
func (sqlis *SQLiteIPAMStorage) Add(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	model := prefixToModel(prefix)
	if model == nil {
		return nil
	}
	exp := false
	if prefix.GetName() != bridgeName && Expirable(ctx) {
		exp = true
	}
	model.Expirable = &exp
	tx := sqlis.DB.Create(model)
	if isCIDRUniquenessViolation(tx.Error) {
		return fmt.Errorf("%w: %v", ErrCIDRConflict, tx.Error)
	}

	return tx.Error
}

// Update -
// Updates or add the database entry.
// Currently, the whole purpose of this function is to update the UpdatedAt
// field in the database that is used by garbage collector logic to clean up
// unused entries that haven't been updated for a long time. And to keep the
// expirable field set for records that can expire based on the context. (GORM
// Save() considers unset entries as well to update the record, thus expirable
// field must be set to either false or true.)
func (sqlis *SQLiteIPAMStorage) Update(ctx context.Context, prefix types.Prefix) error {
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()
	model := prefixToModel(prefix)
	if model == nil {
		return nil
	}
	exp := false
	if prefix.GetName() != bridgeName && Expirable(ctx) {
		exp = true
	}
	model.Expirable = &exp
	tx := sqlis.DB.Save(model)
	if isCIDRUniquenessViolation(tx.Error) {
		return fmt.Errorf("%w: %v", ErrCIDRConflict, tx.Error)
	}
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

// Get finds and returns the first database record matching the given prefix.
// Note: default or unset fields are ignored by the GORM query
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

// Note: When a new column is added to an existing table using AutoMigrate(),
// then by default the DB initializes this column to the 'nul' value depending
// on the type for all existing records. The default value to be used can be
// also specified via the gorm tag 'default'.
func (sqlis *SQLiteIPAMStorage) init() error {
	err := sqlis.DB.AutoMigrate(&Prefix{})
	if err != nil {
		return fmt.Errorf("failed to automigrate: %w", err)
	}
	// Manually set expirable field for any old connection prefixes
	// Note: should not block for too long to avoid risking startup/liveness
	// issues depending on the probe configuration
	if err = sqlis.migrate(context.TODO()); err != nil {
		// Note: No need to return an error. Worst case any old unused prefixes
		// will remain.
		log.Logger.Info("Could not migrate old prefxies", "err", err)
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

// Migrate aims to find prefixes assocaited with connections but created based
// on the old model (before the introduction of the garbage collector).
// Lookup exploits the hierarchy among prefixes of trenches, conduits, worker
// conduits, and connections to identify the connections.
// Old connection prefixes are then updated by setting their expirable fields
// to true, along with implicitly updating their updated_at fields. This way,
// old leaked connection prefixes could be also reaped by the garbage collector
// logic.
// Note: bridge prefixes cannot expire currently
// Note: custom update of old prefixes will set the updated_at fields
func (sqlis *SQLiteIPAMStorage) migrate(ctx context.Context) error {
	var trenchIds []string
	var conduitIds []string
	var workerConduitIds []string
	var err error
	batchSize := 50
	logger := log.Logger.WithValues("func", "migrate")

	findPrefixIds := func(ctx context.Context, query interface{}, args ...interface{}) ([]string, error) {
		var prefixIds []string
		logger.V(1).Info("Find prefixes", "query", query, "args", args)
		select {
		case <-ctx.Done():
			return prefixIds, nil
		default:
			var prefixes []Prefix
			// Fetch all prefixes matching the query in one attempt
			if err := sqlis.DB.Where(query, args...).Find(&prefixes).Error; err != nil {
				return nil, err
			}

			// No matching prefixes in the database
			if len(prefixes) == 0 {
				return nil, nil
			}

			// Save IDs of returned prefixes to return once finished
			for _, prefix := range prefixes {
				prefixIds = append(prefixIds, prefix.Id)
			}
		}
		return prefixIds, nil
	}

	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()

	// Get Trench IDs based on their characteristics not having a parent
	trenchIds, err = findPrefixIds(ctx, "parent_id = ?", "")
	if err != nil || len(trenchIds) == 0 {
		logger.V(1).Info("No Trench IDs retrived", "err", err)
		return err
	}
	logger.Info("Retrieved Trenches", "IDs", trenchIds)

	// Get Conduit IDs by parent ID (which should be a trench ID)
	conduitIds, err = findPrefixIds(ctx, "parent_id IN ?", trenchIds)
	if err != nil || len(conduitIds) == 0 {
		logger.V(1).Info("No Conduit IDs retrieved", "err", err)
		return err
	}
	logger.Info("Retrieved Conduits", "IDs", conduitIds)

	// Get Worker Conduit IDs by parent ID (which should be a Conduit ID)
	workerConduitIds, err = findPrefixIds(ctx, "parent_id IN ?", conduitIds)
	if err != nil || len(workerConduitIds) == 0 {
		logger.V(1).Info("No worker Conduit IDs retrieved", "err", err)
		return err
	}
	logger.Info("Retrieved worker Conduits", "IDs", workerConduitIds)

	// Update records with parent IDs matching any of the Worker Conduit IDs
	// to reflect such records can expire and thus can be subject to garbage
	// collection.
	// Note: Even though records are processed in batches, no need use offsets
	// or pagination, because matching entries are to be updated. Thus, there
	// will be eventually no more records matching the query criteria.
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			tx := sqlis.DB.Begin()
			if tx.Error != nil {
				return tx.Error
			}

			// Get prefixes created according to the old model whose parent ID
			// is among the Worker Conduit IDs.
			// Note: expirable field is not expected to be NULL due to the tag
			// setting a default value false, but better safe than sorry
			var updateEntries []Prefix
			if err := tx.Where("(expirable IS NULL OR expirable = false) AND name != ? AND parent_id IN ?",
				bridgeName, workerConduitIds).Limit(batchSize).Find(&updateEntries).Error; err != nil {
				tx.Rollback()
				return err
			}

			// No more prefixes to update
			if len(updateEntries) == 0 {
				tx.Rollback()
				return nil
			}

			// Update old connection prefixes by setting their expirable field
			// to true, so that the garbage collector apply to them as well.
			// Note: This shall also update the updated_at field.
			for _, entry := range updateEntries {
				logger.V(1).Info("Update prefix", "entry", entry)
				exp := true
				entry.Expirable = &exp
				if err := tx.Save(entry).Error; err != nil {
					tx.Rollback()
					return err
				}
			}

			// Commit the transaction
			if err := tx.Commit().Error; err != nil {
				return err
			}

			logger.Info("Updated prefixes in transaction", "count", len(updateEntries))
		}
	}
}
