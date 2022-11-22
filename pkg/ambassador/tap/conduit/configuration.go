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

//go:generate mockgen -source=configuration.go -destination=mocks/configuration.go -package=mocks
package conduit

import (
	"context"
	"io"
	"sync"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
)

const (
	channelBufferSize = 1
)

type Configuration interface {
	Watch()
	Stop()
}

type configurationImpl struct {
	SetVips                    func([]string) error
	SetStreams                 func([]*nspAPI.Stream)
	Conduit                    *nspAPI.Conduit
	ConfigurationManagerClient nspAPI.ConfigurationManagerClient
	cancel                     context.CancelFunc
	mu                         sync.Mutex
	vipChan                    chan []string
	streamChan                 chan []*nspAPI.Stream
}

func newConfigurationImpl(setVips func([]string) error,
	setStreams func([]*nspAPI.Stream),
	conduit *nspAPI.Conduit,
	configurationManagerClient nspAPI.ConfigurationManagerClient) *configurationImpl {
	c := &configurationImpl{
		SetVips:                    setVips,
		SetStreams:                 setStreams,
		Conduit:                    conduit,
		ConfigurationManagerClient: configurationManagerClient,
	}
	return c
}

func (c *configurationImpl) Watch() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.TODO())
	c.vipChan = make(chan []string, channelBufferSize)
	c.streamChan = make(chan []*nspAPI.Stream, channelBufferSize)
	go c.vipHandler(ctx)
	go c.streamHandler(ctx)
	go c.watchVIPs(ctx)
	go c.watchStreams(ctx)
}

func (c *configurationImpl) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *configurationImpl) vipHandler(ctx context.Context) {
	for {
		select {
		case vips := <-c.vipChan:
			err := c.SetVips(vips)
			if err != nil {
				log.Logger.Error(err, "set vips")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *configurationImpl) streamHandler(ctx context.Context) {
	for {
		select {
		case streams := <-c.streamChan:
			c.SetStreams(streams)
		case <-ctx.Done():
			return
		}
	}
}

func (c *configurationImpl) watchVIPs(ctx context.Context) {
	err := retry.Do(func() error {
		toWatch := &nspAPI.Flow{
			Stream: &nspAPI.Stream{
				Conduit: c.Conduit,
			},
		}
		watchClient, err := c.ConfigurationManagerClient.WatchFlow(ctx, toWatch)
		if err != nil {
			return err
		}
		for {
			response, err := watchClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			// flush previous context in channel
			select {
			case <-c.vipChan:
			default:
			}
			c.vipChan <- flowResponseToVIPSlice(response)
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		log.Logger.Error(err, "watchVIPs") // todo
	}
}

func (c *configurationImpl) watchStreams(ctx context.Context) {
	err := retry.Do(func() error {
		vipsToWatch := &nspAPI.Stream{
			Conduit: c.Conduit,
		}
		watchStreamClient, err := c.ConfigurationManagerClient.WatchStream(ctx, vipsToWatch)
		if err != nil {
			return err
		}
		for {
			streamResponse, err := watchStreamClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			fixStreamsMaxTargets(streamResponse.GetStreams())
			// flush previous context in channel
			select {
			case <-c.streamChan:
			default:
			}
			c.streamChan <- streamResponse.GetStreams()
		}
		return nil
	}, retry.WithContext(ctx),
		retry.WithDelay(500*time.Millisecond),
		retry.WithErrorIngnored())
	if err != nil {
		log.Logger.Error(err, "watchStreams") // todo
	}
}

// fixStreamsMaxTargets fixes the max-target property in the streams received
// by the NSP. NSP clients Version >= to v0.9.0 will wait for the max-target
// property to be received from the NSP, but if NSP used is lower than v0.9.0,
// then this field will not be sent and will be considered as 0.
// To keep everything backward compatible, the value must be 100, not 0.
// Max target has been introduced from commit cca757e1a54f4c19564a1202b88c97f51d8e813b
// (PR 175: https://github.com/Nordix/Meridio/pull/175)
func fixStreamsMaxTargets(streams []*nspAPI.Stream) {
	for _, stream := range streams {
		if stream.GetMaxTargets() <= 0 {
			stream.MaxTargets = 100
		}
	}
}

func flowResponseToVIPSlice(flowResponse *nspAPI.FlowResponse) []string {
	vipMap := map[string]struct{}{}
	for _, flow := range flowResponse.GetFlows() {
		for _, vip := range flow.Vips {
			vipMap[vip.Address] = struct{}{}
		}
	}
	vips := []string{}
	for vip := range vipMap {
		vips = append(vips, vip)
	}
	return vips
}
