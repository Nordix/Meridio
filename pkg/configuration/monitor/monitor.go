/*
Copyright (c) 2021 Nordix Foundation

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

package monitor

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/configuration/reader"
	"github.com/nordix/meridio/pkg/k8s/watcher"
	"github.com/nordix/meridio/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ConfigMapMonitor struct {
	watcher.ObjectMonitorInterface
	ConfigurationRegistry ConfigurationRegistry
	ConfigMapName         string
	Namespace             string
	logger                logr.Logger
}

func New(configMapName string, namespace string, ConfigurationRegistry ConfigurationRegistry) (*ConfigMapMonitor, error) {
	configMapMonitor := &ConfigMapMonitor{
		ConfigMapName:         configMapName,
		Namespace:             namespace,
		ConfigurationRegistry: ConfigurationRegistry,
		logger:                log.Logger.WithValues("class", "ConfigMapMonitor"),
	}

	monitor, err := watcher.NewObjectMonitor(
		context.TODO(),
		configMapName,
		namespace,
		watcher.WatchEventHandler(configMapMonitor),
		func(namespace string) (watcher.WatchObject, error) {
			clientCfg, err := rest.InClusterConfig()
			if err != nil {
				return nil, err
			}

			clientset, err := kubernetes.NewForConfig(clientCfg)
			if err != nil {
				return nil, err
			}

			return clientset.CoreV1().ConfigMaps(namespace), err
		},
	)
	if err != nil {
		return nil, err
	}
	configMapMonitor.ObjectMonitorInterface = monitor

	return configMapMonitor, nil
}

func (cmm *ConfigMapMonitor) Handle(ctx context.Context, event *watch.Event) {
	if event == nil {
		return
	}

	switch event.Type {
	case watch.Added:
		cmm.eventHandler(event)
	case watch.Modified:
		cmm.eventHandler(event)
	case watch.Deleted:
	default:
	}
}

func (cmm *ConfigMapMonitor) End(ctx context.Context, namespace, name string) {}

func (cmm *ConfigMapMonitor) eventHandler(event *watch.Event) {
	configmap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		cmm.logger.Info("Failed to cast event.Object")
		return
	}
	trench, conduits, streams, flows, vips, attractors, gateways, err := reader.UnmarshalConfig(configmap.Data)
	if err != nil {
		cmm.logger.Error(err, "unmarshal")
		return
	}
	trenchConverted, conduitsConverted, streamsConverted, flowsConverted, vipsConverted, attractorsConverted, gatewaysConverted := reader.ConvertAll(
		trench, conduits, streams, flows, vips, attractors, gateways)
	cmm.ConfigurationRegistry.SetTrench(trenchConverted)
	cmm.ConfigurationRegistry.SetConduits(conduitsConverted)
	cmm.ConfigurationRegistry.SetStreams(streamsConverted)
	cmm.ConfigurationRegistry.SetFlows(flowsConverted)
	cmm.ConfigurationRegistry.SetVips(vipsConverted)
	cmm.ConfigurationRegistry.SetAttractors(attractorsConverted)
	cmm.ConfigurationRegistry.SetGateways(gatewaysConverted)
}
