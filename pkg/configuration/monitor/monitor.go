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

	"github.com/nordix/meridio/pkg/configuration/reader"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ConfigMapMonitor struct {
	ConfigurationRegistry ConfigurationRegistry
	ConfigMapName         string
	Namespace             string
	Clientset             *kubernetes.Clientset
	watcher               watch.Interface
}

func New(configMapName string, namespace string, ConfigurationRegistry ConfigurationRegistry) (*ConfigMapMonitor, error) {
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		return nil, err
	}
	configMapMonitor := &ConfigMapMonitor{
		ConfigMapName:         configMapName,
		Namespace:             namespace,
		Clientset:             clientset,
		ConfigurationRegistry: ConfigurationRegistry,
	}
	return configMapMonitor, nil
}

func (cmm *ConfigMapMonitor) Start(context context.Context) {
	for context.Err() == nil {
		var err error
		objectMeta := metav1.ObjectMeta{Name: cmm.ConfigMapName, Namespace: cmm.Namespace}
		cmm.watcher, err = cmm.Clientset.CoreV1().ConfigMaps(cmm.Namespace).Watch(context, metav1.SingleObject(objectMeta))
		if err != nil {
			logrus.Errorf("Unable to watch configmap: %v", err)
			return
		}
		cmm.watchEvent(cmm.watcher.ResultChan())
	}
}

func (cmm *ConfigMapMonitor) watchEvent(eventChannel <-chan watch.Event) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				cmm.eventHandler(&event)
			case watch.Modified:
				cmm.eventHandler(&event)
			case watch.Deleted:
			default:
			}
		} else {
			return
		}
	}
}

func (cmm *ConfigMapMonitor) eventHandler(event *watch.Event) {
	configmap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return
	}
	trench, conduits, streams, flows, vips, attractors, gateways, err := reader.UnmarshalConfig(configmap.Data)
	if err != nil {
		logrus.Errorf("err unmarshal: %v", err)
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
