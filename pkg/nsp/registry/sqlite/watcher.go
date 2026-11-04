/*
Copyright (c) 2026 OpenInfra Foundation Europe

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
	"github.com/nordix/meridio/pkg/nsp/registry/common"
)

// sqliteWatcher removes itself from the registry when stopped. Broadcasts are
// rare (unchanged refreshes skip them), so they cannot be relied on to remove
// stopped watchers: TAPA opens a short Watch on every refresh.
type sqliteWatcher struct {
	*common.RegistryWatcher
	registry *TargetRegistrySQLite
}

// Stop stops the watcher, then removes it from the registry. The watcher lock
// is released before removeWatcher takes the registry lock, so the two are
// never held in the opposite order of notifyAllWatchers (registry lock, then
// watcher lock).
func (w *sqliteWatcher) Stop() {
	w.RegistryWatcher.Stop()
	w.registry.removeWatcher(w.RegistryWatcher)
}
