package watcher

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
)

type monitor struct {
	*ObjectMonitor
	context.CancelFunc
}

// ObjectMonitorManager -
// Allows watching multiple k8s objects of the same type in the same namespace through k8s API
type ObjectMonitorManager struct {
	handler   WatchEventHandler
	monitors  map[string]*monitor
	create    CreateWatchObject
	namespace string
	logger    logr.Logger
}

func NewObjectMonitorManager(ctx context.Context, namespace string, handler WatchEventHandler, create CreateWatchObject) *ObjectMonitorManager {
	return &ObjectMonitorManager{
		monitors:  make(map[string]*monitor),
		namespace: namespace,
		handler:   handler,
		create:    create,
		logger:    log.FromContextOrGlobal(ctx).WithValues("class", "ObjectMonitorManager"),
	}
}

// Manage -
// Manages watchers/monitors based on the input (stops old watchers not included in names)
func (omm *ObjectMonitorManager) Manage(ctx context.Context, names []string) {
	omm.logger.V(1).Info("Manage", "objects", names)

	leftOver := make(map[string]struct{})
	for mkey := range omm.monitors {
		leftOver[mkey] = struct{}{}
	}

	// check for which object to start a monitor
	for _, name := range names {
		if _, ok := omm.monitors[name]; ok {
			// already monitoring
			delete(leftOver, name)
			continue
		}
		om, err := NewObjectMonitor(ctx, name, omm.namespace, omm.handler, omm.create)
		if err != nil {
			omm.logger.Error(err, "Failed to create monitor", "object", name, "namespace", omm.namespace)
		} else {
			ctx, cancel := context.WithCancel(ctx)
			omm.monitors[name] = &monitor{ObjectMonitor: om, CancelFunc: cancel}
			go om.Start(ctx)
		}
	}

	// stop monitoring of objects not included in param names
	for mkey := range leftOver {
		monitor := omm.monitors[mkey]
		monitor.CancelFunc()
		monitor.Stop(context.TODO())
		delete(omm.monitors, mkey)
	}
}
