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

const BridgeName = "bridge" // prefix with name "bridge" is special because it's not periodically updated

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
				time.Time{}, entryThreshHold, BridgeName).Limit(batchSize).Find(&batchOfEligiblePrefixes).Error; err != nil {
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
	if prefix.GetName() != BridgeName && Expirable(ctx) {
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
// Updates or adds the database entry.
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
	if prefix.GetName() != BridgeName && Expirable(ctx) {
		exp = true
	}
	model.Expirable = &exp
	if ok := sqlis.shouldUpdate(ctx, model); !ok {
		return nil // Success, but no-op.
	}
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

// shouldUpdate determines if update is required based on the damping configuration
// or lack of it. In case damping is enabled, it tries to fetch the prefix from the
// db to retrieve its updatedAt timestamp.
func (sqlis *SQLiteIPAMStorage) shouldUpdate(ctx context.Context, model *Prefix) bool {
	updateThreshold, ok := getUpdateDampingThreshold(ctx)
	if !ok {
		return true // Damping not enabled
	}

	var result *Prefix
	// Use Select to specify exactly which fields to load based on ID (due to performance considerations)
	err := sqlis.DB.Model(&Prefix{}).
		Select("id", "cidr", "name", "parent_id", "expirable", "updated_at").
		Where("id = ?", model.Id).
		First(&result).Error
	if err != nil {
		return true // Record might not exist (keep backward compatibility)
	}

	// Explicitly check for significant data changes
	modelExpirable := model.Expirable != nil && *model.Expirable
	resultExpirable := result.Expirable != nil && *result.Expirable
	if model.ParentID != result.ParentID ||
		modelExpirable != resultExpirable ||
		model.Cidr != result.Cidr ||
		model.Name != result.Name {
		return true // Legitimate update changing fields
	}

	// Damping logic
	if time.Since(result.UpdatedAt) < updateThreshold {
		return false // Recently updated within updateThreshold period
	}

	return true
}

// migrate aims to find specific prefixes within the IPAM hierarchy that should
// be subject to garbage collection but currently are not.
//
// It identifies these prefixes by traversing the hierarchy:
// Trenches -> Conduits -> Worker Conduits (nodes) -> Connections.
//
// The method targets two types of prefixes for update:
// 1. Connection prefixes: Children of Worker Conduits, excluding bridge IPs.
// 2. Worker Conduit (node) prefixes: The encapsulating prefixes for connections.
//
// For matching prefixes, their 'expirable' field is set to true. GORM implicitly
// updates their 'updated_at' timestamp during this operation. This change
// allows the garbage collection (GC) logic to identify and remove old or stale
// connection and node prefixes.
//
// Note: Bridge prefixes are intentionally excluded and cannot expire via this migration.
// Note: The 'expirable' field is handled to include prefixes with NULL or false values,
// ensuring proper migration from older schema versions where it might not have been set.
func (sqlis *SQLiteIPAMStorage) migrate(ctx context.Context) error {
	logger := log.Logger.WithValues("func", "migrate")
	sqlis.mu.Lock()
	defer sqlis.mu.Unlock()

	// Find Trench IDs
	var trenchIds []string
	if err := sqlis.DB.WithContext(ctx).Model(&Prefix{}).Where("parent_id = ?", "").Pluck("id", &trenchIds).Error; err != nil {
		return err
	}
	if len(trenchIds) == 0 {
		logger.V(1).Info("No Trench IDs retrieved, skipping migration.")
		return nil // Nothing to do
	}
	logger.Info("Retrieved Trenches for migration", "IDs", trenchIds)

	// Find Conduit IDs
	var conduitIds []string
	if err := sqlis.DB.WithContext(ctx).Model(&Prefix{}).Where("parent_id IN ?", trenchIds).Pluck("id", &conduitIds).Error; err != nil {
		return err
	}
	if len(conduitIds) == 0 {
		logger.V(1).Info("No Conduit IDs retrieved based on trenches, skipping migration for remaining hierarchy.")
		return nil // Nothing to do
	}
	logger.Info("Retrieved Conduits for migration", "IDs", conduitIds)

	// Find Worker Conduit IDs
	var workerConduitIds []string
	if err := sqlis.DB.WithContext(ctx).Model(&Prefix{}).Where("parent_id IN ?", conduitIds).Pluck("id", &workerConduitIds).Error; err != nil {
		return err
	}
	if len(workerConduitIds) == 0 {
		logger.V(1).Info("No Worker Conduit IDs retrieved based on conduits, skipping migration for remaining hierarchy.")
		return nil // Nothing to do
	}
	logger.Info("Retrieved Worker Conduits for migration", "IDs", workerConduitIds)

	// Update the CHILDREN of worker conduits (the "connection" prefixes) so that
	// they could become subject to garbage collection.
	// These are prefixes associated with NSM connections, excluding bridge IPs.
	// Note: expirable field is not expected to be NULL due to the gorm tag
	// setting a default value false, but better safe than sorry
	logger.Info("Updating connection prefixes (children of worker conduits)...")
	resultConnections := sqlis.DB.WithContext(ctx).Model(&Prefix{}).
		Where("parent_id IN ?", workerConduitIds).
		// Exclude bridge prefixes from being marked expirable
		Where("name != ?", BridgeName).
		// Ensure old records with NULL or explicit false are caught
		Where("(expirable IS NULL OR expirable = false)").
		Update("expirable", true)

	if resultConnections.Error != nil {
		return fmt.Errorf("failed to bulk update connection prefixes: %w", resultConnections.Error)
	}
	logger.Info("Successfully migrated connection prefixes.", "updated_count", resultConnections.RowsAffected)

	// Update the WORKER CONDUITS themselves.
	// These are the node-level prefixes that encapsulate connections.
	// Note: expirable field is not expected to be NULL due to the gorm tag
	// setting a default value false, but better safe than sorry
	logger.Info("Updating worker conduit prefixes...")
	resultWorkers := sqlis.DB.WithContext(ctx).Model(&Prefix{}).
		Where("id IN ?", workerConduitIds).
		// Ensure old records with NULL or explicit false are caught
		Where("(expirable IS NULL OR expirable = false)").
		Update("expirable", true)

	if resultWorkers.Error != nil {
		return fmt.Errorf("failed to bulk update worker conduit prefixes: %w", resultWorkers.Error)
	}
	logger.Info("Successfully migrated worker conduit prefixes.", "updated_count", resultWorkers.RowsAffected)

	return nil
}

// --- Test-Specific Exported Methods ---

// NewForTest is a constructor for testing purposes.
// It accepts an already opened *gorm.DB instance. It DOES NOT perform AutoMigrate or
// run init() automatically. This gives tests explicit control over schema setup and
// migration steps.
func NewForTest(db *gorm.DB) (*SQLiteIPAMStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("NewForTest: provided gorm.DB instance is nil")
	}
	sqlis := &SQLiteIPAMStorage{
		DB: db,
	}
	sqlis.prefixTableName = TableNameForModel(sqlis.DB, &Prefix{})
	return sqlis, nil
}

// InitForTest is an exported helper for testing the internal init() logic.
// It performs AutoMigrate of the current Prefix model and runs the migrate function.
func (sqlis *SQLiteIPAMStorage) InitForTest() error {
	return sqlis.init() // Calls the unexported init()
}

// MigrateForTest is an exported helper for testing the internal migrate() logic.
// It provides direct access to the migration function.
func (sqlis *SQLiteIPAMStorage) MigrateForTest(ctx context.Context) error {
	return sqlis.migrate(ctx) // Calls the unexported migrate()
}

func (sqlis *SQLiteIPAMStorage) GetTableNameForTest() string {
	return sqlis.prefixTableName
}

// RunGarbageCollectorOnceForTest is an exported helper for testing
// internal garbageCollector() logic.
func (sqlis *SQLiteIPAMStorage) RunGarbageCollectorOnceForTest(ctx context.Context, threshold time.Duration) error {
	return sqlis.garbageCollector(ctx, threshold)
}
