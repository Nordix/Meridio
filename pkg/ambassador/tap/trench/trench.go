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
	"sync"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/nordix/meridio/pkg/security/credentials"
	"google.golang.org/grpc"
)

type Trench struct {
	Name                       string
	TargetRegistryClient       nspAPI.TargetRegistryClient
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	Namespace                  string
	context                    context.Context
	cancel                     context.CancelFunc
	conduits                   []types.Conduit
	mu                         sync.Mutex
	nspServiceName             string
	nspServicePort             int
	nspConn                    *grpc.ClientConn
}

func New(
	name string,
	namespace string,
	configMapName string,
	nspServiceName string,
	nspServicePort int) (types.Trench, error) {

	context, cancel := context.WithCancel(context.Background())

	trench := &Trench{
		context:        context,
		cancel:         cancel,
		Name:           name,
		Namespace:      namespace,
		conduits:       []types.Conduit{},
		nspServiceName: nspServiceName,
		nspServicePort: nspServicePort,
	}

	err := trench.connectNSPService()
	if err != nil {
		return nil, err
	}

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
	for _, conduit := range t.conduits {
		err := t.RemoveConduit(ctx, conduit)
		if err != nil {
			return err
		}
	}
	t.nspConn.Close()
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

func (t *Trench) GetConduits(conduit *nspAPI.Conduit) []types.Conduit {
	t.mu.Lock()
	defer t.mu.Unlock()
	if conduit == nil {
		return t.conduits
	}
	conduits := []types.Conduit{}
	for _, c := range t.conduits {
		if c.GetStatus() == types.Disconnected || !c.Equals(conduit) {
			continue
		}
		conduits = append(conduits, c)
	}
	return conduits
}

func (t *Trench) GetTargetRegistryClient() nspAPI.TargetRegistryClient {
	return t.TargetRegistryClient
}

func (t *Trench) GetConfigurationManagerClient() nspAPI.ConfigurationManagerClient {
	return t.ConfigurationManagerClient
}

func (t *Trench) Equals(trench *nspAPI.Trench) bool {
	if trench == nil {
		return true
	}
	name := true
	if trench.GetName() != "" {
		name = t.GetName() == trench.GetName()
	}
	return name
}

func (t *Trench) getIndex(conduitName string) int {
	for i, conduit := range t.conduits {
		if conduit.GetName() == conduitName {
			return i
		}
	}
	return -1
}

func (t *Trench) connectNSPService() error {
	var err error
	t.nspConn, err = grpc.Dial(t.getNSPService(),
		grpc.WithTransportCredentials(
			credentials.GetClient(context.Background()),
		),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil
	}

	t.TargetRegistryClient = nspAPI.NewTargetRegistryClient(t.nspConn)
	t.ConfigurationManagerClient = nspAPI.NewConfigurationManagerClient(t.nspConn)
	return nil
}

func (t *Trench) getNSPService() string {
	return nsp.GetServiceName(t.nspServiceName, t.GetName(), t.GetNamespace(), t.nspServicePort)
}
