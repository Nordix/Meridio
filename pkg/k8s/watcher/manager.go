package watcher

import (
	"context"

	"github.com/sirupsen/logrus"
)

type monitor struct {
	*Monitor
	context.CancelFunc
}

// MonitorManager -
// Allows watching multiple k8s objects of the same type in the same namespace through k8s API
type MonitorManager struct {
	handler   WatchEventHandler
	monitors  map[string]*monitor
	create    CreateWatchObject
	namespace string
}

func NewObjectMonitorManager(namespace string, handler WatchEventHandler, create CreateWatchObject) *MonitorManager {
	return &MonitorManager{
		monitors:  make(map[string]*monitor),
		namespace: namespace,
		handler:   handler,
		create:    create,
	}
}

// Manage -
// Manages watchers/monitors based on the input (stops old watchers not included in names)
func (mm *MonitorManager) Manage(ctx context.Context, names []string) {
	logrus.Debugf("Manage monitoring of objects (namespace=%s): %v", mm.namespace, names)

	leftOver := make(map[string]struct{})
	for mkey := range mm.monitors {
		leftOver[mkey] = struct{}{}
	}

	// check for which object to start a monitor
	for _, name := range names {
		if _, ok := mm.monitors[name]; ok {
			// already monitoring
			delete(leftOver, name)
			continue
		}
		om, err := NewObjectMonitor(name, mm.namespace, mm.handler, mm.create)
		if err != nil {
			logrus.Errorf("Failed to create object monitor %v (namespace=%s); %v", name, mm.namespace, err)
		} else {
			ctx, cancel := context.WithCancel(ctx)
			mm.monitors[name] = &monitor{Monitor: om, CancelFunc: cancel}
			go om.Start(ctx)
		}
	}

	// stop monitoring of objects not included in param names
	for mkey := range leftOver {
		monitor := mm.monitors[mkey]
		monitor.CancelFunc()
		monitor.Stop(context.TODO())
		delete(mm.monitors, mkey)
	}
}
