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

package configuration

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nordix/meridio/pkg/configuration/reader"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Watch Meridio-Operator spawned configmap

type OperatorWatcher struct {
	configMap   string
	namespace   string
	clientset   *kubernetes.Clientset
	configEvent chan<- *OperatorConfig
	watcher     watch.Interface
	context     context.Context
	cancel      context.CancelFunc
}

type OperatorConfig struct {
	GWs  *reader.GatewayList
	VIPs *reader.VipList
}

func (oc *OperatorConfig) String() string {
	return fmt.Sprintf("{GWs:%v, VIPs:%v}", oc.GWs, oc.VIPs)
}

func (w *OperatorWatcher) Start() {
	for w.context.Err() == nil {
		var err error
		w.watcher, err = w.clientset.CoreV1().ConfigMaps(w.namespace).Watch(context.TODO(),
			metav1.SingleObject(metav1.ObjectMeta{Name: w.configMap, Namespace: w.namespace}))
		if err != nil {
			logrus.Errorf("Unable to watch configmap: %v", err)
			return
		}
		w.watchEvent(w.watcher.ResultChan())
	}
}

func (w *OperatorWatcher) Delete() {
	w.cancel()
	w.watcher.Stop()
}

func (w *OperatorWatcher) eventHandler(event *watch.Event) {
	configmap, ok := event.Object.(*corev1.ConfigMap)
	if !ok {
		return
	}

	c := &OperatorConfig{
		GWs:  &reader.GatewayList{},
		VIPs: &reader.VipList{},
	}
	var err error
	c.GWs.Gateways, err = reader.UnmarshalGateways(configmap.Data[reader.GatewaysConfigKey])
	if err != nil {
		logrus.Errorf("err unmarshal: %v", err)
		return
	}
	c.VIPs.Vips, err = reader.UnmarshalVips(configmap.Data[reader.VipsConfigKey])
	if err != nil {
		logrus.Errorf("err unmarshal: %v", err)
		return
	}
	logrus.Debugf("watcher: %v", c)
	w.configEvent <- c
}

func (w *OperatorWatcher) watchEvent(eventChannel <-chan watch.Event) {
	for {
		event, open := <-eventChannel
		if open {
			switch event.Type {
			case watch.Added:
				w.eventHandler(&event)
			case watch.Modified:
				w.eventHandler(&event)
			case watch.Deleted:
			default:
			}
		} else {
			return
		}
	}
}

func NewOperatorWatcher(configMap string, namespace string, configEvent chan<- *OperatorConfig) *OperatorWatcher {
	clientCfg, err := rest.InClusterConfig()
	if err != nil {
		logrus.Errorf("Unable to get InCluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(clientCfg)
	if err != nil {
		logrus.Errorf("Unable to create clientset: %v", err)
	}

	context, cancel := context.WithCancel(context.Background())

	watcher := &OperatorWatcher{
		configMap:   configMap,
		namespace:   namespace,
		clientset:   clientset,
		configEvent: configEvent,
		context:     context,
		cancel:      cancel,
	}
	logrus.Debugf("NewOperatorWatcher: name: %v, ns: %v", configMap, namespace)
	return watcher
}

// DiffOperatorConfigItem -
// returns true if different
func DiffOperatorConfigItem(a, b interface{}) bool {
	switch a := a.(type) {
	case *reader.GatewayList:
		if b, ok := b.(*reader.GatewayList); ok {
			return DiffGatewayConfig(a, b)
		} else {
			// not the same type
			logrus.Warnf("DiffOperatorConfigItem: type mismatch")
			return true
		}
	case *reader.VipList:
		if b, ok := b.(*reader.VipList); ok {
			return DiffVipConfig(a, b)
		} else {
			logrus.Warnf("DiffOperatorConfigItem: type mismatch")
			return true
		}
	default:
		logrus.Fatalf("DiffOperatorConfigItem: unknown format")
		return false
	}
}

// DiffGatewayConfig -
// returns true if different
func DiffGatewayConfig(a, b *reader.GatewayList) bool {
	if len(a.Gateways) != len(b.Gateways) {
		// different length
		return true
	}

	mapA := makeMapFromGWList(a.Gateways)
	mapB := makeMapFromGWList(b.Gateways)
	return func() bool {
		for name := range mapA {
			if _, ok := mapB[name]; !ok {
				return true
			}
		}
		for name := range mapB {
			if _, ok := mapA[name]; !ok {
				return true
			}
		}
		for key, value := range mapA {
			if !reflect.DeepEqual(mapB[key], value) {
				return true
			}
		}
		return false
	}()
}

// DiffVipConfig -
// returns true if different
func DiffVipConfig(a, b *reader.VipList) bool {
	if len(a.Vips) != len(b.Vips) {
		// different length
		return true
	}

	mapA := makeMapFromVipList(a.Vips)
	mapB := makeMapFromVipList(b.Vips)
	return func() bool {
		for name := range mapA {
			if _, ok := mapB[name]; !ok {
				return true
			}
		}
		for name := range mapB {
			if _, ok := mapA[name]; !ok {
				return true
			}
		}
		for key, value := range mapA {
			if !reflect.DeepEqual(mapB[key], value) {
				return true
			}
		}
		return false
	}()
}

// AddrListFromVipConfig -
// Generate string list of VIP addresses based on the config
func AddrListFromVipConfig(vips *reader.VipList) []string {
	list := []string{}
	for _, item := range vips.Vips {
		list = append(list, item.Address)
	}
	return list
}

// Input is a slice of Vips.
// Return a map with key as vip names.
func makeMapFromVipList(lst []*reader.Vip) map[string]reader.Vip {
	ret := make(map[string]reader.Vip)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}

// Input is a slice of Gateways.
// Return a map with key as gateway names.
func makeMapFromGWList(lst []*reader.Gateway) map[string]reader.Gateway {
	ret := make(map[string]reader.Gateway)
	for _, item := range lst {
		ret[item.Name] = *item
	}
	return ret
}
