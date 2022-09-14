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

	"github.com/sirupsen/logrus"
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
}

func NewDatabase(ctx context.Context, channel chan<- struct{}) *Database {
	return &Database{
		storage: make(map[databaseKey]map[string][]byte),
		channel: channel,
		ctx:     ctx,
	}
}

// Load -
// Searches the database for key in secret referenced by name.
func (sd *Database) Load(namespace, name, key string) ([]byte, error) {
	dbKey := databaseKey{name: name, namespace: namespace}
	sd.rm.RLock()
	defer sd.rm.RUnlock()
	if dbVal, ok := sd.storage[dbKey]; ok {
		if val, ok := dbVal[key]; ok {
			return val, nil
		}
		return nil, fmt.Errorf("key %v not found in %s", key, dbKey)
	}
	return nil, fmt.Errorf("%s not found", dbKey)
}

// Handle -
// Handles update of database based on the event Object.
func (sd *Database) Handle(ctx context.Context, event *watch.Event) {
	if event.Type == watch.Error {
		logrus.Debugf("Database: ERROR event; %s", event.Object)
		return
	}

	secret, ok := event.Object.(*corev1.Secret)
	if !ok {
		logrus.Errorf("Database: FAILED to cast event.Object to %T", &corev1.Secret{})
		return
	}

	logrus.Tracef("Database: event (%s)", event.Type)
	switch event.Type {
	case watch.Added:
		fallthrough
	case watch.Modified:
		sd.update(secret)
	case watch.Deleted:
		sd.delete(secret)
	default:
	}
}

// End -
// Removes secret with namespace and name from database, and signals change.
// Note: monitoring of particular secret was ordered to stop, because it is
// no longer of interest. Thus there's no point keeping related information.
func (sd *Database) End(ctx context.Context, namespace, name string) {
	dbKey := databaseKey{name: name, namespace: namespace}
	ok := false
	logrus.Debugf("Database: End %s", dbKey)

	sd.rm.Lock()
	if _, ok = sd.storage[dbKey]; ok {
		sd.storage[dbKey] = nil
	}
	sd.rm.Unlock()

	if ok {
		sd.signal()
	}
}

// update -
// Overwrites matching database entry with secret, and signals change.
func (sd *Database) update(secret *corev1.Secret) {
	dbVal := make(map[string][]byte)
	dbKey := databaseKey{name: secret.Name, namespace: secret.Namespace}
	logrus.Debugf("Database: update %s", dbKey)

	for key, val := range secret.Data {
		dbVal[key] = val
	}

	sd.rm.Lock()
	sd.storage[dbKey] = dbVal
	sd.rm.Unlock()

	sd.signal()
}

// delete -
// Deletes matching database entry representing a secret, and signals change.
func (sd *Database) delete(secret *corev1.Secret) {
	dbKey := databaseKey{name: secret.Name, namespace: secret.Namespace}
	ok := false
	logrus.Debugf("Database: delete %s", dbKey)

	sd.rm.Lock()
	if _, ok = sd.storage[dbKey]; ok {
		sd.storage[dbKey] = nil
	}
	sd.rm.Unlock()

	if ok {
		sd.signal()
	}
}

// signal -
// Indicates that database has changed.
func (sd *Database) signal() {
	if sd.channel != nil {
		select {
		case sd.channel <- struct{}{}:
		case <-sd.ctx.Done():
		}
	}
}
