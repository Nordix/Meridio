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

package target

import (
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	targetAPI "github.com/nordix/meridio/api/target/v1"
	"github.com/nordix/meridio/pkg/target/types"
)

type conduitWatcher struct {
	watcher        targetAPI.Ambassador_WatchConduitServer
	conduitToWatch *nspAPI.Conduit
}

func (cw *conduitWatcher) notify(trench types.Trench) {
	if cw.watcher == nil {
		return
	}
	response := &nspAPI.ConduitResponse{
		Conduits: []*nspAPI.Conduit{},
	}
	if trench == nil {
		_ = cw.watcher.Send(response)
		return
	}
	conduits := trench.GetConduits(cw.conduitToWatch)
	for _, conduit := range conduits {
		c := &nspAPI.Conduit{
			Name: conduit.GetName(),
			Trench: &nspAPI.Trench{
				Name: trench.GetName(),
			},
		}
		response.Conduits = append(response.Conduits, c)
	}
	_ = cw.watcher.Send(response)
}

type streamWatcher struct {
	watcher       targetAPI.Ambassador_WatchStreamServer
	streamToWatch *nspAPI.Stream
}

func (sw *streamWatcher) notify(trench types.Trench) {
	if sw.watcher == nil {
		return
	}
	response := &nspAPI.StreamResponse{
		Streams: []*nspAPI.Stream{},
	}
	if trench == nil {
		_ = sw.watcher.Send(response)
		return
	}
	conduits := trench.GetConduits(nil)
	for _, conduit := range conduits {
		streams := conduit.GetStreams(sw.streamToWatch)
		for _, stream := range streams {
			s := &nspAPI.Stream{
				Name: stream.GetName(),
				Conduit: &nspAPI.Conduit{
					Name: conduit.GetName(),
					Trench: &nspAPI.Trench{
						Name: trench.GetName(),
					},
				},
			}
			response.Streams = append(response.Streams, s)
		}
	}
	_ = sw.watcher.Send(response)
}
