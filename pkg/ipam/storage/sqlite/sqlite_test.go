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

package sqlite_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/storage/sqlite"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	sqliteDrv "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// --- Constants and Helper Structs ---

const (
	dbFileName = "test.db" // Used for file-based tests
	bridgeName = sqlite.BridgeName
)

// OldPrefix represents a very old pre-GC schema for migration tests
type OldPrefix struct {
	Id       string `gorm:"primaryKey"`
	Name     string
	Cidr     string
	ParentID string
	Parent   *OldPrefix
}

// TableName tells GORM to use "prefixes" as the table name for OldPrefix.
// This ensures consistency with the production Prefix model's table name.
func (OldPrefix) TableName() string {
	return "prefixes"
}

// --- Test Setup/Teardown Helpers for In-Memory DB ---

// setupTestDB creates an in-memory GORM DB connection for tests.
// It returns the DB instance and a cleanup function.
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	dbConn, err := gorm.Open(sqliteDrv.Open(":memory:"), &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time { return time.Now().UTC() }, // Ensures UTC for consistency
	})
	assert.NoError(t, err, "Failed to open in-memory database connection")

	// Get *sql.DB from gorm.DB and ensure it's closed
	sqlDB, err := dbConn.DB()
	assert.NoError(t, err, "Failed to get underlying SQL DB")

	cleanup := func() {
		if sqlDB != nil {
			assert.NoError(t, sqlDB.Close(), "Failed to close database connection")
		}
	}
	return dbConn, cleanup
}

// newSQLiteIPAMStorageForTest creates a SQLiteIPAMStorage instance for testing
// using a provided GORM DB connection.
func newSQLiteIPAMStorageForTest(t *testing.T, db *gorm.DB) *sqlite.SQLiteIPAMStorage {
	sqlis, err := sqlite.NewForTest(db)
	assert.NoError(t, err, "Failed to create SQLiteIPAMStorage for test")
	assert.NotNil(t, sqlis, "SQLiteIPAMStorage instance should not be nil")
	return sqlis
}

// --- Core IPAM Storage Functionality Tests (using file-based DB) ---

func Test_Add_Get(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	assert.NotNil(t, store)
	err = store.Add(context.Background(), p1)
	assert.Nil(t, err)
	pGet, err := store.Get(context.Background(), "abc", nil)
	assert.Nil(t, err)
	assert.Equal(t, p1, pGet)

	p2 := prefix.New("abc", "192.168.0.0/28", p1)
	err = store.Add(context.Background(), p2)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "abc", nil)
	assert.Nil(t, err)
	assert.Equal(t, p1, pGet)
	pGet, err = store.Get(context.Background(), "abc", p1)
	assert.Nil(t, err)
	assert.Equal(t, p2, pGet)

	p3 := prefix.New("def", "192.168.0.0/30", p2)
	err = store.Add(context.Background(), p3)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "def", p2)
	assert.Nil(t, err)
	assert.Equal(t, p3, pGet)

	p4 := prefix.New("ghi", "192.168.0.0/32", p3)
	err = store.Add(context.Background(), p4)
	assert.Nil(t, err)
	pGet, err = store.Get(context.Background(), "ghi", p3)
	assert.Nil(t, err)
	assert.Equal(t, p4, pGet)

	p5 := prefix.New("abc", "192.168.0.1/32", nil)
	err = store.Add(context.Background(), p5)
	assert.NotNil(t, err)
}

func Test_GetChilds(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/16", nil)
	_ = store.Add(context.Background(), p1)
	p2 := prefix.New("abc", "192.168.0.0/24", p1)
	_ = store.Add(context.Background(), p2)
	p3 := prefix.New("def", "192.168.1.0/24", p1)
	_ = store.Add(context.Background(), p3)
	p4 := prefix.New("def", "192.168.1.0/32", p3)
	_ = store.Add(context.Background(), p4)

	childs, err := store.GetChilds(context.Background(), p1)
	assert.Nil(t, err)
	assert.Len(t, childs, 2)
	assert.Contains(t, childs, p2)
	assert.Contains(t, childs, p3)

	childs, err = store.GetChilds(context.Background(), p2)
	assert.Nil(t, err)
	assert.Len(t, childs, 0)

	childs, err = store.GetChilds(context.Background(), p3)
	assert.Nil(t, err)
	assert.Len(t, childs, 1)
	assert.Contains(t, childs, p4)
}

func Test_Delete(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/24", nil)
	_ = store.Add(context.Background(), p1)
	p2 := prefix.New("abc", "192.168.0.0/32", p1)
	_ = store.Add(context.Background(), p2)
	p3 := prefix.New("def", "192.168.0.1/32", p1)
	_ = store.Add(context.Background(), p3)
	err = store.Delete(context.Background(), p2)
	assert.Nil(t, err)
	childs, _ := store.GetChilds(context.Background(), p1)
	assert.Len(t, childs, 1)
	assert.Contains(t, childs, p3)

	err = store.Delete(context.Background(), p1)
	assert.Nil(t, err)
	pGet, _ := store.Get(context.Background(), "abc", nil)
	assert.Nil(t, pGet)
	pGet, _ = store.Get(context.Background(), "def", p1)
	assert.Nil(t, pGet)
}

func Test_CIDR_Uniqueness_Constraint(t *testing.T) {
	_ = os.Remove(dbFileName)
	defer os.Remove(dbFileName)

	store, err := sqlite.New(dbFileName)
	assert.Nil(t, err)

	p1 := prefix.New("abc", "192.168.0.0/16", nil)
	err = store.Add(context.Background(), p1)
	assert.Nil(t, err)
	p2 := prefix.New("abc", "192.168.0.0/24", p1)
	err = store.Add(context.Background(), p2)
	assert.Nil(t, err)

	p3 := prefix.New("def", "192.168.0.0/24", p1)
	err = store.Add(context.Background(), p3)
	assert.NotNil(t, err, "Should not be possible adding CIDR %s twice for the same parent", p3.GetCidr())
	assert.True(t, errors.Is(err, sqlite.ErrCIDRConflict))
	p3Get, _ := store.Get(context.Background(), p3.GetName(), p3.GetParent())
	assert.Nil(t, p3Get)

	err = store.Update(context.Background(), p3)
	assert.NotNil(t, err, "Should not be possible adding CIDR %s twice for the same parent by creating a new entry with update", p3.GetCidr())
	assert.True(t, errors.Is(err, sqlite.ErrCIDRConflict))
	p3Get, _ = store.Get(context.Background(), p3.GetName(), p3.GetParent())
	assert.Nil(t, p3Get)

	p4 := prefix.New("def", "192.168.0.0/32", p2)
	err = store.Add(context.Background(), p4)
	assert.Nil(t, err)

	p5 := prefix.New("xyz", "192.168.0.1/32", p2)
	err = store.Add(context.Background(), p5)
	assert.Nil(t, err)
	p6 := prefix.New("xyz", "192.168.0.0/32", p2)
	err = store.Update(context.Background(), p6)
	assert.NotNil(t, err, "Should not be possible adding CIDR %s twice for the same parent by updating an existing entry", p6.GetCidr())
	assert.True(t, errors.Is(err, sqlite.ErrCIDRConflict))

	childs, err := store.GetChilds(context.Background(), p1)
	assert.Nil(t, err)
	assert.Len(t, childs, 1)
	assert.Contains(t, childs, p2)

	childs, err = store.GetChilds(context.Background(), p2)
	assert.Nil(t, err)
	assert.Len(t, childs, 2)
	assert.Contains(t, childs, p4)
	assert.Contains(t, childs, p5)
}

// --- Migration Tests (using in-memory DB) ---

// TestStandardInitWithNewForTest demonstrates testing the full init() flow
func TestStandardInitWithNewForTest(t *testing.T) {
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()

	sqlis := newSQLiteIPAMStorageForTest(t, dbConn)

	// Now call the exported InitForTest to run the full automigrate + migrate sequence
	err := sqlis.InitForTest()
	assert.NoError(t, err, "InitForTest should complete successfully")

	// Assert final state after a full init (e.g., table exists).
	// This test primarily validates that InitForTest doesn't error and sets up the schema.
	var p sqlite.Prefix
	assert.NoError(t, dbConn.Table(sqlis.GetTableNameForTest()).Limit(1).Find(&p).Error, "Prefix table should be created")
}

// TestMigrateFromOldSchema tests migration from a very old pre-GC schema (no timestamps, no expirable)
func TestMigrateFromOldSchema(t *testing.T) {
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Simulate Very Old Schema: AutoMigrate with OldPrefix
	assert.NoError(t, dbConn.AutoMigrate(&OldPrefix{}), "Failed to auto-migrate old schema")

	// 2. Seed Data into Old Schema (list parent entries before their children)
	rawSeedData := []*OldPrefix{
		{Id: "trench-a-0", Name: "trench-a-0", Cidr: "10.0.0.0/16"},
		{Id: "load-balancer-a1-trench-a-0", ParentID: "trench-a-0", Name: "load-balancer-a1", Cidr: "10.0.0.0/20"},
		{Id: "load-balancer-a2-trench-a-0", ParentID: "trench-a-0", Name: "load-balancer-a2", Cidr: "10.0.16.0/20"},
		{Id: "worker1-load-balancer-a1-trench-a-0", ParentID: "load-balancer-a1-trench-a-0", Name: "worker1", Cidr: "10.0.1.0/24"},
		{Id: "worker2-load-balancer-a1-trench-a-0", ParentID: "load-balancer-a1-trench-a-0", Name: "worker2", Cidr: "10.0.2.0/24"},
		{Id: "connection-1-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: "connection-1", Cidr: "10.0.1.2/32"},
		{Id: "connection-2-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: "connection-2", Cidr: "10.0.1.3/32"},
		{Id: "connection-3-worker2-load-balancer-a1-trench-a-0", ParentID: "worker2-load-balancer-a1-trench-a-0", Name: "connection-3", Cidr: "10.0.2.2/32"},
		{Id: "bridge-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: bridgeName, Cidr: "10.0.1.1/32"},
		{Id: "bridge-worker2-load-balancer-a1-trench-a-0", ParentID: "worker2-load-balancer-a1-trench-a-0", Name: bridgeName, Cidr: "10.0.2.1/32"},
	}
	// Set Parent reference for each Prefix with a parent ID
	createdPrefixes := make(map[string]*OldPrefix)
	for _, seed := range rawSeedData {
		if seed.ParentID != "" {
			parent, ok := createdPrefixes[seed.ParentID]
			assert.True(t, ok, "Parent with ID '%s' not found for old prefix '%s'. Ensure parents are listed before children.", seed.ParentID, seed.Id)
			seed.Parent = parent // Assign the object pointer
		}
		assert.NoError(t, dbConn.Create(seed).Error, fmt.Sprintf("Failed to create old prefix %s", seed.Id))
		createdPrefixes[seed.Id] = seed // Store object for future children
	}

	// Reference time for checking UpdatedAt later, after the new timestamp and expirable columns are added.
	refTime := time.Now().UTC()
	time.Sleep(10 * time.Millisecond) // Ensure any subsequently added/populated `UpdatedAt` values are measurably after this point.

	// --- INSPECTION POINT 1: After old schema seeded ---
	t.Log("--- Inspection Point 1: After old schema seeded (no timestamps/expirable) ---")
	// Fetch all prefixes again, explicitly Preloading the "Parent" association
	var allPrefixesAfterSeeding []OldPrefix
	assert.NoError(t, dbConn.Preload("Parent").Find(&allPrefixesAfterSeeding).Error, "Failed to fetch all old prefixes after seeding with preload")
	for _, p := range allPrefixesAfterSeeding {
		// Now, p.Parent should be populated if it has a ParentID
		t.Logf("Seeded data: %+v, parent ID: %s, parent obj: %+v", p, p.ParentID, p.Parent)
		if p.ParentID != "" {
			assert.NotNil(t, p.Parent, "Parent should not be nil for prefix %s", p.Id)
			assert.Equal(t, p.ParentID, p.Parent.Id, "Parent ID mismatch for %s", p.Id)
		} else {
			assert.Nil(t, p.Parent, "Root prefix %s should have a nil parent", p.Id)
		}
	}

	// 3. Run AutoMigrate with the CURRENT Prefix struct (adds CreatedAt, UpdatedAt, Expirable, unique index)
	assert.NoError(t, dbConn.AutoMigrate(&sqlite.Prefix{}), "Failed to auto-migrate to current schema")

	// --- INSPECTION POINT 2: After current AutoMigrate, before custom migrate() ---
	t.Log("--- Inspection Point 2: After current AutoMigrate, before custom migrate() ---")
	// Fetch all prefixes again to get their state after the AutoMigrate to the new schema
	var allPrefixesAfterAutoMigrate []sqlite.Prefix
	assert.NoError(t, dbConn.Find(&allPrefixesAfterAutoMigrate).Error, "Failed to fetch all prefixes after AutoMigrate")

	// Verify entries after AutoMigrate: Expirable should be false (default value via GORM tag), UpdatedAt should be set.
	timestampsAfterAutoMigrate := make(map[string]time.Time) // Will store these for later comparison
	for _, p := range allPrefixesAfterAutoMigrate {
		t.Logf("  Prefix %s after AutoMigrate: Expirable=%v, UpdatedAt=%v", p.Id, p.Expirable, p.UpdatedAt)
		assert.NotNil(t, p.Expirable, "Expirable should exist for %s after AutoMigrate", p.Id)
		assert.False(t, *p.Expirable, "Expirable should be false for %s after AutoMigrate (default)", p.Id)
		assert.True(t, p.UpdatedAt.IsZero(), "UpdatedAt should be zero for %s after AutoMigrate and before custom migrate", p.Id)
		timestampsAfterAutoMigrate[p.Id] = p.UpdatedAt // This will store the zero time for later comparison
	}

	time.Sleep(10 * time.Millisecond) // Small delay for `UpdatedAt` checks

	// 4. Create SQLiteIPAMStorage instance based on the existing db and execute the custom migrate method
	sqlis, err := sqlite.NewForTest(dbConn)
	assert.NoError(t, err, "Failed to create SQLiteIPAMStorage for test")
	ctx := context.Background()
	err = sqlis.MigrateForTest(ctx)
	assert.NoError(t, err, "MigrateForTest method returned an error")

	// --- INSPECTION POINT 3: Final state after custom migrate() ---
	t.Log("--- Inspection Point 3: Final state after custom migrate() ---")
	// Verify the final state (node and connection prefixes should be updated)
	expectedUpdatedIDs := map[string]bool{
		"worker1-load-balancer-a1-trench-a-0":              true,
		"worker2-load-balancer-a1-trench-a-0":              true,
		"connection-1-worker1-load-balancer-a1-trench-a-0": true,
		"connection-2-worker1-load-balancer-a1-trench-a-0": true,
		"connection-3-worker2-load-balancer-a1-trench-a-0": true,
	}

	for id := range expectedUpdatedIDs {
		var p sqlite.Prefix
		assert.NoError(t, dbConn.Where("id = ?", id).First(&p).Error, "Failed to find updated prefix %s", id)
		assert.NotNil(t, p.Expirable)
		assert.True(t, *p.Expirable, "Expirable should be true for %s", id)
		// UpdatedAt should now have changed from the zero timestamp after AutoMigrate
		assert.False(t, p.UpdatedAt.IsZero(), "UpdatedAt should be set to a non-zero value for %s after migrate", id)
		// It should be after refTime (which was captured before AutoMigrate)
		assert.True(t, p.UpdatedAt.After(refTime), "UpdatedAt should be after refTime for %s after migrate", id)
	}

	// Verify prefixes that should NOT be updated by migrate
	notUpdatedIDs := []string{
		"trench-a-0",
		"load-balancer-a1-trench-a-0",
		"load-balancer-a2-trench-a-0",
		"bridge-worker1-load-balancer-a1-trench-a-0",
		"bridge-worker2-load-balancer-a1-trench-a-0",
	}
	for _, id := range notUpdatedIDs {
		var p sqlite.Prefix
		assert.NoError(t, dbConn.Where("id = ?", id).First(&p).Error)
		// Expirable should remain false (default from AutoMigrate, not affected by migrate logic)
		assert.NotNil(t, p.Expirable)
		assert.False(t, *p.Expirable, "%s should NOT be expirable", id)
		// UpdatedAt should NOT change from the zero timestamp after AutoMigrate, it remains zero.
		assert.True(t, p.UpdatedAt.IsZero(), "%s UpdatedAt should remain zero after migrate", id)
	}
}

// TestMigrateWithSpecificInitialState tests the migration from a specific initial state
// where some expirable values are explicitly set and one UpdatedAt is zero.
// Simulates an upgrade from the current Prefix model to a version where node prefix removal
// support is introduced by the GC logic.
func TestMigrateWithSpecificInitialState(t *testing.T) {
	dbConn, cleanup := setupTestDB(t) // Using the helper func
	defer cleanup()

	// 1. AutoMigrate with the CURRENT Prefix struct (full schema)
	assert.NoError(t, dbConn.AutoMigrate(&sqlite.Prefix{}), "Failed to auto-migrate current schema")

	// 2. Seed Data with specific expirable and UpdatedAt values
	falseVal := false
	trueVal := true
	rawSeedData := []*sqlite.Prefix{
		// Trench (not target for migrate, expirable false)
		{Id: "trench-a-0", Name: "trench-a-0", Cidr: "10.0.0.0/16", Expirable: &falseVal},
		// Load Balancers (not targeted by migrate, expirable false)
		{Id: "load-balancer-a1-trench-a-0", ParentID: "trench-a-0", Name: "load-balancer-a1", Cidr: "10.0.0.0/20", Expirable: &falseVal},
		{Id: "load-balancer-a2-trench-a-0", ParentID: "trench-a-0", Name: "load-balancer-a2", Cidr: "10.0.16.0/20", Expirable: &falseVal},
		// Workers (targeted by migrate, expirable false initially -> true after migrate)
		{Id: "worker1-load-balancer-a1-trench-a-0", ParentID: "load-balancer-a1-trench-a-0", Name: "worker1", Cidr: "10.0.1.0/24", Expirable: &falseVal},
		{Id: "worker2-load-balancer-a1-trench-a-0", ParentID: "load-balancer-a1-trench-a-0", Name: "worker2", Cidr: "10.0.2.0/24", Expirable: &falseVal},
		// Connections (expirable true initially, thus should NOT be impacted by migrate)
		{Id: "connection-1-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: "connection-1", Cidr: "10.0.1.2/32", Expirable: &trueVal},
		{Id: "connection-2-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: "connection-2", Cidr: "10.0.1.3/32", Expirable: &trueVal},
		{Id: "connection-3-worker2-load-balancer-a1-trench-a-0", ParentID: "worker2-load-balancer-a1-trench-a-0", Name: "connection-3", Cidr: "10.0.2.2/32", Expirable: &trueVal},
		// Bridges (not targeted by migrate, expirable false)
		{Id: "bridge-worker1-load-balancer-a1-trench-a-0", ParentID: "worker1-load-balancer-a1-trench-a-0", Name: bridgeName, Cidr: "10.0.1.1/32", Expirable: &falseVal},
		{Id: "bridge-worker2-load-balancer-a1-trench-a-0", ParentID: "worker2-load-balancer-a1-trench-a-0", Name: bridgeName, Cidr: "10.0.2.1/32", Expirable: &falseVal},
	}

	createdPrefixes := make(map[string]*sqlite.Prefix)
	for _, seed := range rawSeedData {
		if seed.ParentID != "" {
			parent, ok := createdPrefixes[seed.ParentID]
			assert.True(t, ok, "Parent with ID '%s' not found for prefix '%s'. Ensure parents are listed before children.", seed.ParentID, seed.Id)
			seed.Parent = parent // Assign the object pointer
		}
		assert.NoError(t, dbConn.Create(seed).Error, fmt.Sprintf("Failed to create prefix %s", seed.Id))
		createdPrefixes[seed.Id] = seed // Store object for future children
	}

	// Manually set 'trench-a-0's UpdatedAt to zero to simulate a very old record
	// that didn't have a valid timestamp (e.g., inherited from an old model based db).
	// We need to fetch it first, then update only the UpdatedAt field.
	var trenchA0 sqlite.Prefix
	assert.NoError(t, dbConn.Where("id = ?", "trench-a-0").First(&trenchA0).Error)
	assert.NoError(t, dbConn.Model(&trenchA0).Update("UpdatedAt", time.Time{}).Error, "Failed to manually set UpdatedAt to zero")

	// Capture initial timestamps for verification for ALL prefixes AFTER modifications
	initialTimestamps := make(map[string]time.Time)
	var allPrefixesInitial []sqlite.Prefix
	assert.NoError(t, dbConn.Find(&allPrefixesInitial).Error)
	for _, p := range allPrefixesInitial {
		initialTimestamps[p.Id] = p.UpdatedAt
	}
	time.Sleep(10 * time.Millisecond) // Small delay before calling migrate()

	// --- INSPECTION POINT 1: After seeding and manual update, before custom migrate() ---
	t.Log("--- Inspection Point 1: After seeding and manual update, before custom migrate() ---")

	// Verify initial states after setup
	var checkPrefixes []sqlite.Prefix
	assert.NoError(t, dbConn.Find(&checkPrefixes).Error)
	for _, p := range checkPrefixes {
		t.Logf("  Prefix %s (Initial): Expirable=%v, UpdatedAt=%v", p.Id, p.Expirable, p.UpdatedAt)
		assert.NotNil(t, p.Expirable, "Expirable should be set for %s", p.Id)

		// Determine expected Expirable based on initial seed data (connections are true, others false)
		if strings.HasPrefix(p.Name, "connection-") {
			assert.True(t, *p.Expirable, "Connection prefix %s should be expirable true initially", p.Id)
		} else {
			assert.False(t, *p.Expirable, "%s should be expirable false initially", p.Id)
		}

		// Determine expected UpdatedAt based on initial seed data
		if p.Id == "trench-a-0" {
			assert.True(t, p.UpdatedAt.IsZero(), "trench-a-0 UpdatedAt should be zero initially", p.Id)
		} else {
			assert.False(t, p.UpdatedAt.IsZero(), "%s UpdatedAt should be non-zero initially", p.Id)
		}
	}

	// 3. Create SQLiteIPAMStorage instance and execute the migrate method
	sqlis := newSQLiteIPAMStorageForTest(t, dbConn)
	ctx := context.Background()
	err := sqlis.MigrateForTest(ctx)
	assert.NoError(t, err, "MigrateForTest method returned an error")

	// --- INSPECTION POINT 2: Final state after custom migrate() ---
	t.Log("--- Inspection Point 2: Final state after custom migrate() ---")

	// Define which IDs should be updated by the migrate function
	// Only 'worker' prefixes should have their 'Expirable' set to true AND 'UpdatedAt' refreshed.
	// Connections, having started as 'true', should NOT be in this list.
	expectedUpdatedIDs := map[string]bool{
		"worker1-load-balancer-a1-trench-a-0": true,
		"worker2-load-balancer-a1-trench-a-0": true,
	}

	// Define IDs of prefixes that are connections (used for specific Expirable check)
	connectionIDs := map[string]bool{
		"connection-1-worker1-load-balancer-a1-trench-a-0": true,
		"connection-2-worker1-load-balancer-a1-trench-a-0": true,
		"connection-3-worker2-load-balancer-a1-trench-a-0": true,
	}

	// Verify all prefixes after migration
	var allPrefixesFinal []sqlite.Prefix
	assert.NoError(t, dbConn.Find(&allPrefixesFinal).Error)

	for _, p := range allPrefixesFinal {
		t.Logf("  Prefix %s (Final): Expirable=%v, UpdatedAt=%v", p.Id, p.Expirable, p.UpdatedAt)

		if expectedUpdatedIDs[p.Id] {
			// These are 'worker' prefixes: Expirable becomes true, UpdatedAt changes
			assert.NotNil(t, p.Expirable)
			assert.True(t, *p.Expirable, "Expirable should be true for %s after migrate", p.Id)
			assert.True(t, p.UpdatedAt.After(initialTimestamps[p.Id]), "UpdatedAt should be updated for %s after migrate", p.Id)
		} else {
			// These are trenches, LBs, bridges, AND connections
			assert.NotNil(t, p.Expirable)

			if connectionIDs[p.Id] {
				// Connections: Expirable should remain true, UpdatedAt should remain unchanged
				assert.True(t, *p.Expirable, "%s (connection) Expirable should remain true after migrate", p.Id)
				assert.True(t, p.UpdatedAt.Equal(initialTimestamps[p.Id]), "UpdatedAt for %s (connection) should NOT change after migrate", p.Id)
			} else {
				// Trenches, LBs, Bridges: Expirable should remain false, UpdatedAt should remain unchanged
				assert.False(t, *p.Expirable, "Expirable should remain false for %s after migrate", p.Id)
				// Check UpdatedAt: trench-a-0 should remain zero, others should remain their initial non-zero time.
				if p.Id == "trench-a-0" {
					assert.True(t, p.UpdatedAt.IsZero(), "trench-a-0 UpdatedAt should remain zero after migrate", p.Id)
				} else {
					assert.True(t, p.UpdatedAt.Equal(initialTimestamps[p.Id]), "UpdatedAt for %s should NOT change after migrate", p.Id)
				}
			}
		}
	}
}

func TestGarbageCollectorLogic(t *testing.T) {
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()
	assert.NoError(t, dbConn.AutoMigrate(&sqlite.Prefix{}), "Failed to auto-migrate schema")

	sqlis := newSQLiteIPAMStorageForTest(t, dbConn)
	ctx := context.Background()

	// Define GC threshold for the test.
	gcThreshold := 15 * time.Minute // Prefixes older than 15 minutes should be collected

	// Define timestamps relative to the threshold for precise control.
	now := time.Now().UTC()
	pastThreshold := now.Add(-gcThreshold - 1*time.Minute) // 1 minute older than the threshold
	recentTime := now.Add(-1 * time.Minute)                // 1 minute ago, well within the threshold

	falseVal := false
	trueVal := true

	// Define all prefixes to be seeded, along with their expected state after GC runs.
	// Order is important: Parents must appear before their children.
	rawSeedData := []struct {
		Id              string
		Name            string
		Cidr            string
		ParentID        string
		Expirable       *bool
		UpdatedAt       time.Time // Explicitly setting UpdatedAt for the test
		ExpectedAfterGC bool      // true if prefix should exist after GC, false if deleted
	}{
		// --- Trench (Non-Expirable, has no parent) ---
		{Id: "trench-a-0", Name: "trench-a-0", Cidr: "10.0.0.0/16", ParentID: "", Expirable: &falseVal, UpdatedAt: pastThreshold, ExpectedAfterGC: true},

		// --- Conduit (Non-Expirable, parent Trench) ---
		{Id: "load-balancer-a1-trench-a-0", Name: "load-balancer-a1", Cidr: "10.0.0.0/20", ParentID: "trench-a-0", Expirable: &falseVal, UpdatedAt: pastThreshold, ExpectedAfterGC: true},

		// --- Scenario 1: Stale Worker (anchor) and its children -> DELETED ---
		// Worker (Expirable) - Stale, Expirable: Anchor for deletion
		{Id: "worker1-load-balancer-a1-trench-a-0", Name: "worker1", Cidr: "10.0.1.0/24", ParentID: "load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: pastThreshold, ExpectedAfterGC: false},
		// Connection (Expirable) - Deleted with parent
		{Id: "connection-1-worker1-load-balancer-a1-trench-a-0", Name: "connection-1", Cidr: "10.0.1.2/32", ParentID: "worker1-load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: pastThreshold, ExpectedAfterGC: false},
		// Bridge (Non-Expirable) - Deleted with parent
		{Id: "bridge-worker1-load-balancer-a1-trench-a-0", Name: sqlite.BridgeName, Cidr: "10.0.1.1/32", ParentID: "worker1-load-balancer-a1-trench-a-0", Expirable: &falseVal, UpdatedAt: pastThreshold, ExpectedAfterGC: false},

		// --- Scenario 2: Expirable, Recent Worker and its children ---
		// Worker (Expirable) - Recent, Expirable: Remains
		{Id: "worker2-load-balancer-a1-trench-a-0", Name: "worker2", Cidr: "10.0.2.0/24", ParentID: "load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: recentTime, ExpectedAfterGC: true},
		// Connection (Expirable) - Recent, Expirable: Remains (both connection and parent are recent)
		{Id: "connection-2-worker2-load-balancer-a1-trench-a-0", Name: "connection-2", Cidr: "10.0.2.2/32", ParentID: "worker2-load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: recentTime, ExpectedAfterGC: true},
		// Connection (Expirable) - Stale, Expirable: DELETED by GC
		{Id: "connection-3-worker2-load-balancer-a1-trench-a-0", Name: "connection-3", Cidr: "10.0.2.3/32", ParentID: "worker2-load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: pastThreshold, ExpectedAfterGC: false},
		// Bridge (Non-Expirable) - Remains as worker2 remains
		{Id: "bridge-worker2-load-balancer-a1-trench-a-0", Name: sqlite.BridgeName, Cidr: "10.0.2.1/32", ParentID: "worker2-load-balancer-a1-trench-a-0", Expirable: &falseVal, UpdatedAt: pastThreshold, ExpectedAfterGC: true},

		// --- Scenario 3: Non-expirable, Stale Worker and its children ---
		// Worker (Non-Expirable) - Stale, Non-Expirable: Remains
		{Id: "worker3-load-balancer-a1-trench-a-0", Name: "worker3", Cidr: "10.0.3.0/24", ParentID: "load-balancer-a1-trench-a-0", Expirable: &falseVal, UpdatedAt: pastThreshold, ExpectedAfterGC: true},
		// Connection (Expirable) - Stale, Expirable: Deleted by GC
		{Id: "connection-4-worker3-load-balancer-a1-trench-a-0", Name: "connection-4", Cidr: "10.0.3.4/32", ParentID: "worker3-load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: pastThreshold, ExpectedAfterGC: false},
		// Connection (Expirable) - Recent, Expirable: Remains (parent is non-expirable and connection is recent)
		{Id: "connection-5-worker3-load-balancer-a1-trench-a-0", Name: "connection-5", Cidr: "10.0.3.5/32", ParentID: "worker3-load-balancer-a1-trench-a-0", Expirable: &trueVal, UpdatedAt: recentTime, ExpectedAfterGC: true},
	}

	// 2. Seed the database using the rawSeedData and the hierarchical approach
	createdPrefixes := make(map[string]*sqlite.Prefix)
	for _, seed := range rawSeedData {
		p := sqlite.Prefix{
			Id:        seed.Id,
			Name:      seed.Name,
			Cidr:      seed.Cidr,
			Expirable: seed.Expirable,
			UpdatedAt: seed.UpdatedAt, // GORM will respect this if non-zero
		}

		if seed.ParentID != "" {
			parent, ok := createdPrefixes[seed.ParentID]
			assert.True(t, ok, "Parent with ID '%s' not found for prefix '%s'. Ensure parents are listed before children.", seed.ParentID, p.Id)
			p.Parent = parent          // Assign the object pointer
			p.ParentID = seed.ParentID // Keep ParentID populated for the column
		}

		assert.NoError(t, dbConn.Create(&p).Error, fmt.Sprintf("Failed to create prefix %s", p.Id))
		createdPrefixes[p.Id] = &p // Store the created object for future children
	}

	t.Log("--- Initial state before GC ---")
	var allPrefixesInitial []sqlite.Prefix
	assert.NoError(t, dbConn.Preload("Parent").Find(&allPrefixesInitial).Error)
	//assert.NoError(t, dbConn.Find(&allPrefixesInitial).Error)
	assert.Equal(t, len(allPrefixesInitial), len(createdPrefixes))
	for _, p := range allPrefixesInitial {
		t.Logf("  Prefix %s (Name: %s): Expirable=%v, UpdatedAt=%v (IsZero=%v), ParentID=%s, Parent obj: %+v",
			p.Id, p.Name, p.Expirable, p.UpdatedAt, p.UpdatedAt.IsZero(), p.ParentID, p.Parent)
		cp, ok := createdPrefixes[p.Id]
		assert.True(t, ok)
		assert.Equal(t, cp.UpdatedAt, p.UpdatedAt)
	}

	// 3. Run the garbage collector method once
	err := sqlis.RunGarbageCollectorOnceForTest(ctx, gcThreshold)
	assert.NoError(t, err, "Garbage collector returned an error")

	t.Log("--- Final state after GC ---")
	var allPrefixesFinal []sqlite.Prefix
	assert.NoError(t, dbConn.Find(&allPrefixesFinal).Error)

	// Create a map of found IDs for easy lookup
	foundPrefixes := make(map[string]struct{})
	for _, p := range allPrefixesFinal {
		foundPrefixes[p.Id] = struct{}{}
		t.Logf("  Prefix %s (Name: %s): Expirable=%v, UpdatedAt=%v (IsZero=%v), ParentID=%s",
			p.Id, p.Name, p.Expirable, p.UpdatedAt, p.UpdatedAt.IsZero(), p.ParentID)
	}

	// 4. Assertions: Verify the presence/absence of each prefix as expected
	for _, seed := range rawSeedData {
		_, found := foundPrefixes[seed.Id] // Use the pre-calculated ID for lookup
		if seed.ExpectedAfterGC {
			assert.True(t, found, "Prefix %s was expected to exist after GC but was NOT found", seed.Id)
		} else {
			assert.False(t, found, "Prefix %s was expected to be DELETED by GC but was found", seed.Id)
		}
	}
}

func TestUpdateDampingUpdatedAt(t *testing.T) {
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()
	assert.NoError(t, dbConn.AutoMigrate(&sqlite.Prefix{}), "Failed to auto-migrate schema")

	sqlis := newSQLiteIPAMStorageForTest(t, dbConn)
	baseCtx := context.Background()

	// Define common test prefix properties
	prefixID := "testNode"
	prefixCidr := "192.168.0.0/24"

	// Helper to create and insert a prefix with a specific UpdatedAt
	createTestPrefix := func(id string, updatedAt time.Time) types.Prefix {
		expirableTrue := false // false to avoid the usage of sqlite.WithExpirable for sqlite.Update
		p := &sqlite.Prefix{
			ParentID:  "", // no parent so that Prefix Id and Prefix Name could be the same values
			Id:        id, // mimic prefixToModel()
			Name:      id, // mimic prefixToModel()
			Cidr:      prefixCidr,
			Expirable: &expirableTrue,
			UpdatedAt: updatedAt,
		}
		assert.NoError(t, dbConn.Create(p).Error, fmt.Sprintf("Failed to create initial prefix for test, id: %q, cidr: %q", id, prefixCidr))
		return prefix.New(p.Name, p.Cidr, nil) // Convert to types.Prefix for storage's Update method
	}

	// Helper to fetch UpdatedAt directly from DB
	fetchUpdatedAt := func(id string) time.Time {
		var p sqlite.Prefix
		err := dbConn.Select("updated_at").Where("id = ?", id).First(&p).Error
		assert.NoError(t, err, fmt.Sprintf("Failed to fetch UpdatedAt for ID: %q", id))
		return p.UpdatedAt
	}

	deletePrefixById := func(id string) {
		_ = dbConn.Where("id = ?", id).Delete(&sqlite.Prefix{}).Error
	}

	dampingThreshold := 1 * time.Minute

	// Scenario 1: Damping enabled, recently updated (within threshold), no other changes -> UpdatedAt should NOT change
	t.Run("UpdatedAt_Damped", func(t *testing.T) {
		pID := prefixID + "Damped"
		defer deletePrefixById(pID)
		initialUpdatedAt := time.Now().UTC().Add(-30 * time.Second) // 30 seconds ago
		testPrefix := createTestPrefix(pID, initialUpdatedAt)

		err := sqlis.Update(sqlite.WithUpdateDamping(baseCtx, dampingThreshold), testPrefix)
		assert.NoError(t, err, "Update should not error when damped")

		finalUpdatedAt := fetchUpdatedAt(pID)
		assert.True(t, finalUpdatedAt.Equal(initialUpdatedAt), "UpdatedAt should remain unchanged (damped)")
	})

	// Scenario 2: Damping enabled, updated long ago (older than threshold), no other changes -> UpdatedAt should CHANGE
	t.Run("UpdatedAt_NotDamped_OldPrefix", func(t *testing.T) {
		pID := prefixID + "NotDampedOldPrefix"
		defer deletePrefixById(pID)
		initialUpdatedAt := time.Now().UTC().Add(-2 * time.Minute) // 2 minutes ago
		testPrefix := createTestPrefix(pID, initialUpdatedAt)

		err := sqlis.Update(sqlite.WithUpdateDamping(baseCtx, dampingThreshold), testPrefix)
		assert.NoError(t, err, "Update should not error when not damped")

		finalUpdatedAt := fetchUpdatedAt(pID)
		assert.True(t, finalUpdatedAt.After(initialUpdatedAt), "UpdatedAt should be newer than initial time")
	})

	// Scenario 3: Damping disabled (no context), recently updated -> UpdatedAt should CHANGE
	t.Run("UpdatedAt_NotDamped_Disabled", func(t *testing.T) {
		pID := prefixID + "NotDampedDisabled"
		defer deletePrefixById(pID)
		initialUpdatedAt := time.Now().UTC().Add(-30 * time.Second) // 30 seconds ago
		testPrefix := createTestPrefix(pID, initialUpdatedAt)

		// No damping context applied
		err := sqlis.Update(baseCtx, testPrefix)
		assert.NoError(t, err, "Update should not error when damping is disabled")

		finalUpdatedAt := fetchUpdatedAt(pID)
		assert.True(t, finalUpdatedAt.After(initialUpdatedAt), "UpdatedAt should be newer than initial time")
	})

	// Scenario 4: Update a non-existent record (should create it, damping logic won't prevent initial creation)
	t.Run("UpdatedAt_NonExistentRecord", func(t *testing.T) {
		pID := prefixID + "NonExistent"
		defer deletePrefixById(pID)
		prefixName := pID // prefixToModel() in sqlite.Update() sets ID based on prefix Name

		// Create a types.Prefix representing the new record
		newPrefix := prefix.New(prefixName, prefixCidr, nil)

		err := sqlis.Update(sqlite.WithUpdateDamping(baseCtx, dampingThreshold), newPrefix)
		assert.NoError(t, err, "Update should succeed for a non-existent record (effectively a create)")

		// Verify the record was created and UpdatedAt is set
		var fetchedPrefix sqlite.Prefix
		err = dbConn.Where("id = ?", pID).First(&fetchedPrefix).Error
		assert.NoError(t, err, "Failed to fetch newly created prefix")
		assert.False(t, fetchedPrefix.UpdatedAt.IsZero(), "UpdatedAt should be set for a new record")
		assert.Equal(t, pID, fetchedPrefix.Id)
		assert.Equal(t, prefixCidr, fetchedPrefix.Cidr)
		assert.Equal(t, prefixName, fetchedPrefix.Name)
	})

	// Scenario 5: Update recent record changing its CIDR (should update it, damping logic won't prevent legitimate update)
	t.Run("UpdatedAt_NotDamped_RecentPrefixChanged", func(t *testing.T) {
		pID := prefixID + "RecentPrefixChanged"
		defer deletePrefixById(pID)
		prefixName := pID // prefixToModel() in sqlite.Update() sets ID based on prefix Name

		initialUpdatedAt := time.Now().UTC().Add(-30 * time.Second) // 30 seconds ago
		testPrefix := createTestPrefix(pID, initialUpdatedAt)

		changedPrefixCidr := "192.168.1.0/24"
		changedTestPrefix := prefix.New(testPrefix.GetName(), changedPrefixCidr, testPrefix.GetParent())
		err := sqlis.Update(sqlite.WithUpdateDamping(baseCtx, dampingThreshold), changedTestPrefix)
		assert.NoError(t, err, "Update should not error when damped")

		var fetchedPrefix sqlite.Prefix
		err = dbConn.Where("id = ?", pID).First(&fetchedPrefix).Error
		assert.NoError(t, err, "Failed to fetch updated prefix")
		assert.True(t, fetchedPrefix.UpdatedAt.After(initialUpdatedAt), "UpdatedAt should be newer than initial time")
		assert.NotEqual(t, fetchedPrefix.Cidr, testPrefix.GetCidr(), "Cidr should be updated")
		assert.Equal(t, prefixName, fetchedPrefix.Name)
	})
}

func TestExpirable(t *testing.T) {
	dbConn, cleanup := setupTestDB(t)
	defer cleanup()
	assert.NoError(t, dbConn.AutoMigrate(&sqlite.Prefix{}), "Failed to auto-migrate schema")

	sqlis := newSQLiteIPAMStorageForTest(t, dbConn)
	baseCtx := context.Background()

	// Define common test prefix properties
	commonPrefixName := "testNode"
	commonPrefixCidr := "192.168.0.0/24"
	commonParentID := ""

	assertPrefixData := func(t *testing.T, name, expectedCidr, expectedParentID string, expectedExpirable bool, msg string) *sqlite.Prefix {
		var fetched []sqlite.Prefix
		err := dbConn.Where("name = ?", name).Find(&fetched).Error
		assert.NoError(t, err, fmt.Sprintf("%s: Error fetching prefix with Name %q", msg, name))
		assert.Len(t, fetched, 1, fmt.Sprintf("%s: Expected exactly one prefix with Name %q, but found %d", msg, name, len(fetched)))
		resultPrefix := fetched[0]

		assert.False(t, resultPrefix.UpdatedAt.IsZero(), fmt.Sprintf("%s: UpdatedAt should be set for prefix with Name %q", msg, name))
		assert.Equal(t, expectedCidr, resultPrefix.Cidr, fmt.Sprintf("%s: CIDR mismatch for prefix with Name %q", msg, name))
		assert.Equal(t, expectedParentID, resultPrefix.ParentID, fmt.Sprintf("%s: ParentID mismatch for prefix with Name %q", msg, name))
		if expectedExpirable {
			assert.True(t, resultPrefix.Expirable != nil && *resultPrefix.Expirable,
				fmt.Sprintf("%s: Expirable should be true for prefix with Name %q", msg, name))
		} else {
			assert.True(t, resultPrefix.Expirable != nil && !*resultPrefix.Expirable,
				fmt.Sprintf("%s: Expirable should be false for prefix with Name %q", msg, name))
		}
		return &resultPrefix
	}

	t.Run("InitialCreate_NonExpirable", func(t *testing.T) {
		testName := commonPrefixName + "InitialNonExpirable"
		testPrefix := prefix.New(testName, commonPrefixCidr, nil)

		defer func() {
			_ = sqlis.Delete(baseCtx, testPrefix)
		}()

		err := sqlis.Add(baseCtx, testPrefix)
		assert.NoError(t, err, "Add should succeed for initial creation")
		assertPrefixData(t, testName, commonPrefixCidr, commonParentID, false, "After initial create")
	})

	t.Run("InitialCreate_Expirable", func(t *testing.T) {
		testName := commonPrefixName + "InitialExpirable"
		testPrefix := prefix.New(testName, commonPrefixCidr, nil)

		defer func() {
			_ = sqlis.Delete(baseCtx, testPrefix)
		}()

		err := sqlis.Add(sqlite.WithExpirable(baseCtx), testPrefix)
		assert.NoError(t, err, "Add should succeed for initial creation")
		assertPrefixData(t, testName, commonPrefixCidr, commonParentID, true, "After initial create")
	})

	t.Run("UpdateExisting_WithExpirable", func(t *testing.T) {
		testName := commonPrefixName + "ExpirableUpdate"
		testPrefix := prefix.New(testName, commonPrefixCidr, nil)

		defer func() {
			_ = sqlis.Delete(baseCtx, testPrefix)
		}()

		err := sqlis.Update(baseCtx, testPrefix)
		assert.NoError(t, err, "Update should succeed for a non-existent record (effectively a create)")
		assertPrefixData(t, testName, commonPrefixCidr, commonParentID, false, "Before expirable update")

		err = sqlis.Update(sqlite.WithExpirable(baseCtx), testPrefix)
		assert.NoError(t, err, "Update should succeed with expirable context")
		assertPrefixData(t, testName, commonPrefixCidr, commonParentID, true, "After expirable update")
	})
}
