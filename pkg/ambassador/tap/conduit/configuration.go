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

package conduit

import (
	"context"
	"io"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ambassador/tap/types"
	"github.com/sirupsen/logrus"
)

type Configuration struct {
	Conduit                    types.Conduit
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	Cancel                     context.CancelFunc
}

type SetVips interface {
	SetVIPs([]string) error
}

func NewConfiguration(conduit types.Conduit, configurationManagerClient nspAPI.ConfigurationManagerClient) *Configuration {
	c := &Configuration{
		Conduit:                    conduit,
		ConfigurationManagerClient: configurationManagerClient,
	}
	return c
}

func (c *Configuration) WatchVIPs(ctx context.Context) {
	ctx, c.Cancel = context.WithCancel(ctx)
	for {
		vipsToWatch := &nspAPI.Vip{
			Trench: &nspAPI.Trench{
				Name: c.Conduit.GetTrench().GetName(),
			},
		}
		watchVIPClient, err := c.ConfigurationManagerClient.WatchVip(ctx, vipsToWatch)
		if err != nil {
			logrus.Warnf("err watchVIPClient.Recv: %v", err) // todo
			continue
		}
		for {
			vipResponse, err := watchVIPClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				logrus.Warnf("err watchVIPClient.Recv: %v", err) // todo
				break
			}
			err = c.Conduit.SetVIPs(vipResponse.ToSlice())
			if err != nil {
				logrus.Warnf("err set vips: %v", err) // todo
			}
		}
	}
}

func (c *Configuration) Delete() {
	if c.Cancel != nil {
		c.Cancel()
	}
}
