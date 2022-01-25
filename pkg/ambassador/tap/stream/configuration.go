/*
Copyright (c) 2021-2022 Nordix Foundation

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

package stream

import (
	"context"
	"io"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/sirupsen/logrus"
)

type Configuration interface {
	WatchStream(ctx context.Context)
}

type configurationImpl struct {
	Watcher                    watcher
	Stream                     *nspAPI.Stream
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
}

type watcher interface {
	StreamExists(bool) error
}

func newConfigurationImpl(watcher watcher,
	stream *nspAPI.Stream,
	configurationManagerClient nspAPI.ConfigurationManagerClient) *configurationImpl {
	c := &configurationImpl{
		Watcher:                    watcher,
		Stream:                     stream,
		ConfigurationManagerClient: configurationManagerClient,
	}
	return c
}

func (c *configurationImpl) WatchStream(ctx context.Context) {
	for { // Todo: retry
		if ctx.Err() != nil {
			return
		}
		watchStreamClient, err := c.ConfigurationManagerClient.WatchStream(ctx, c.Stream)
		if err != nil {
			logrus.Warnf("err WatchStream: %v", err) // todo
			continue
		}
		for {
			streamResponse, err := watchStreamClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				logrus.Warnf("err watchStreamClient.Recv: %v", err) // todo
				break
			}
			err = c.Watcher.StreamExists(len(streamResponse.Streams) > 0)
			if err != nil {
				logrus.Warnf("err set vips: %v", err) // todo
			}
		}
	}
}
