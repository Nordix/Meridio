/*
Copyright (c) 2022 Nordix Foundation

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

package secret

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// key used by Database composed of the name of the secret and its namespace
type databaseKey struct {
	name      string // name of the secret object
	namespace string // namespace of the secret object
}

func (dbk databaseKey) String() string {
	return fmt.Sprintf("secret %s namespace %s", dbk.name, dbk.namespace)
}

// Database -
// Stores data of multiple Secret objects.
// Synchronizes with Secret objects through watcher.WatchEventHandler interface.
type Database struct {
	storage map[databaseKey]map[string][]byte
	channel chan<- struct{} // write to channel to indicate change in storage
	rm      sync.RWMutex
	ctx     context.Context
	logger  logr.Logger
}

func NewDatabase(ctx context.Context, channel chan<- struct{}) *Database {
	return &Database{
		storage: make(map[databaseKey]map[string][]byte),
		channel: channel,
		ctx:     ctx,
		logger:  log.FromContextOrGlobal(ctx).WithValues("class", "Database"),
	}
}

// Load -
// Searches the database for key in secret referenced by name.
func (db *Database) Load(namespace, name, key string) ([]byte, error) {
	dbKey := databaseKey{name: name, namespace: namespace}
	db.rm.RLock()
	defer db.rm.RUnlock()
	if dbVal, ok := db.storage[dbKey]; ok {
		if val, ok := dbVal[key]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("key %v not found in %s", key, dbKey)
	}
	return nil, fmt.Errorf("%s not found", dbKey)
}

// Handle -
// Handles update of database based on the event Object.
func (db *Database) Handle(ctx context.Context, event *watch.Event) {
	if event.Type == watch.Error {
		db.logger.V(1).Info("Handle Error event", "event", event.Object)
		return
	}

	secret, ok := event.Object.(*corev1.Secret)
	if !ok {
		db.logger.Error(fmt.Errorf("cast failed"), "unexpected event object", "event", event.Object)
		return
	}

	db.logger.V(2).Info("Handle", "type", event.Type)
	switch event.Type {
	case watch.Added:
		fallthrough
	case watch.Modified:
		db.update(secret)
	case watch.Deleted:
		db.delete(secret)
	default:
	}
}

// End -
// Removes secret with namespace and name from database, and signals change.
// Note: monitoring of particular secret was ordered to stop, because it is
// no longer of interest. Thus there's no point keeping related information.
func (db *Database) End(ctx context.Context, namespace, name string) {
	dbKey := databaseKey{name: name, namespace: namespace}
	ok := false
	db.logger.V(1).Info("End", "key", dbKey)

	db.rm.Lock()
	if _, ok = db.storage[dbKey]; ok {
		delete(db.storage, dbKey)
	}
	db.rm.Unlock()

	if ok {
		db.signal()
	}
}

// update -
// Overwrites matching database entry with secret, and signals change.
func (db *Database) update(secret *corev1.Secret) {
	dbVal := make(map[string][]byte)
	dbKey := databaseKey{name: secret.Name, namespace: secret.Namespace}
	db.logger.V(1).Info("update", "key", dbKey)

	for key, val := range secret.Data {
		dbVal[key] = val
	}

	db.rm.Lock()
	db.storage[dbKey] = dbVal
	db.rm.Unlock()

	db.signal()
}

// delete -
// Deletes matching database entry representing a secret, and signals change.
func (db *Database) delete(secret *corev1.Secret) {
	dbKey := databaseKey{name: secret.Name, namespace: secret.Namespace}
	ok := false
	db.logger.V(1).Info("delete", "key", dbKey)

	db.rm.Lock()
	if _, ok = db.storage[dbKey]; ok {
		delete(db.storage, dbKey)
	}
	db.rm.Unlock()

	if ok {
		db.signal()
	}
}

// signal -
// Indicates that database has changed.
func (db *Database) signal() {
	if db.channel != nil {
		select {
		case db.channel <- struct{}{}:
		case <-db.ctx.Done():
		}
	}
}
