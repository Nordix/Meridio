package watcher

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type ObjectMonitorInterface interface {
	Start(ctx context.Context)
	Stop(ctx context.Context)
}

type ObjectMonitorManagerInterface interface {
	Manage(ctx context.Context, names []string)
}

// WatchObject -
// Is a wrapper around Watch (implemented by v1.SecretInterface, v1.PodInterface etc.)
type WatchObject interface {
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// CreateWatchObject -
// A function that creates a WatchObject to be watched (monitored)
type CreateWatchObject func(namespace string) (WatchObject, error)

// WatchEventHandler -
// Handles watch events relayed by Monitor
// TODO: check if End() could be changed to accept *watch.Event
type WatchEventHandler interface {
	Handle(ctx context.Context, event *watch.Event)
	End(ctx context.Context, namespace, name string)
}
