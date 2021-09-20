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

package trench

import (
	"context"
	"errors"
	"fmt"
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp"
	"github.com/nordix/meridio/pkg/configuration"
	"github.com/nordix/meridio/pkg/target/types"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Trench struct {
	Name                 string
	NSPClient            nspAPI.NetworkServicePlateformServiceClient
	Namespace            string
	context              context.Context
	cancel               context.CancelFunc
	conduits             []types.Conduit
	vips                 []string
	configurationWatcher *configuration.OperatorWatcher
	configWatcher        <-chan *configuration.OperatorConfig
	mu                   sync.Mutex
	nspServiceName       string
	nspServicePort       int
	nspConn              *grpc.ClientConn
}

func New(
	name string,
	namespace string,
	configMapName string,
	nspServiceName string,
	nspServicePort int) (types.Trench, error) {

	configMapName = fmt.Sprintf("%s-%s", configMapName, name)
	configWatcher := make(chan *configuration.OperatorConfig, 10)
	configurationWatcher := configuration.NewOperatorWatcher(configMapName, namespace, configWatcher)
	go configurationWatcher.Start()

	context, cancel := context.WithCancel(context.Background())

	trench := &Trench{
		context:              context,
		cancel:               cancel,
		Name:                 name,
		Namespace:            namespace,
		conduits:             []types.Conduit{},
		vips:                 []string{},
		configurationWatcher: configurationWatcher,
		configWatcher:        configWatcher,
		nspServiceName:       nspServiceName,
		nspServicePort:       nspServicePort,
	}

	err := trench.connectNSPService()
	if err != nil {
		return nil, err
	}

	go trench.watchConfig()

	return trench, nil
}

func (t *Trench) GetName() string {
	return t.Name
}

func (t *Trench) GetNamespace() string {
	return t.Namespace
}

func (t *Trench) Delete(ctx context.Context) error {
	t.cancel()
	t.nspConn.Close()
	t.configurationWatcher.Delete()
	for _, conduit := range t.conduits {
		err := t.RemoveConduit(ctx, conduit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Trench) AddConduit(ctx context.Context, conduit types.Conduit) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	index := t.getIndex(conduit.GetName())
	if index >= 0 {
		return errors.New("this conduit is already connected")
	}
	err := conduit.Connect(ctx)
	if err != nil {
		return err
	}
	err = conduit.SetVIPs(t.vips)
	if err != nil {
		return err // todo: disconnect
	}
	t.conduits = append(t.conduits, conduit)
	return nil
}

func (t *Trench) RemoveConduit(ctx context.Context, conduit types.Conduit) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	index := t.getIndex(conduit.GetName())
	if index < 0 {
		return errors.New("this conduit is not connected")
	}
	err := conduit.Disconnect(ctx)
	if err != nil {
		return nil
	}
	t.conduits = append(t.conduits[:index], t.conduits[index+1:]...)
	return nil
}

func (t *Trench) GetConduit(conduitName string) types.Conduit {
	t.mu.Lock()
	defer t.mu.Unlock()
	index := t.getIndex(conduitName)
	if index < 0 {
		return nil
	}
	return t.conduits[index]
}

func (t *Trench) GetConduits() []types.Conduit {
	return t.conduits
}

func (t *Trench) GetNSPClient() nspAPI.NetworkServicePlateformServiceClient {
	return t.NSPClient
}

func (t *Trench) getIndex(conduitName string) int {
	for i, conduit := range t.conduits {
		if conduit.GetName() == conduitName {
			return i
		}
	}
	return -1
}

func (t *Trench) watchConfig() {
	for {
		select {
		case config := <-t.configWatcher:
			t.vips = configuration.AddrListFromVipConfig(config.VIPs)
			for _, conduit := range t.conduits {
				err := conduit.SetVIPs(t.vips)
				if err != nil {
					logrus.Errorf("Error updating to VIPs for the conduit: %V", err)
				}
			}
		case <-t.context.Done():
			return
		}
	}
}

func (t *Trench) connectNSPService() error {
	var err error
	t.nspConn, err = grpc.Dial(t.getNSPService(), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil
	}

	t.NSPClient = nspAPI.NewNetworkServicePlateformServiceClient(t.nspConn)
	return nil
}

func (t *Trench) getNSPService() string {
	return fmt.Sprintf("%s-%s.%s:%d", t.nspServiceName, t.GetName(), t.GetNamespace(), t.nspServicePort)
}
