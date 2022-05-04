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

package keepalive

import (
	"context"
	"time"

	"github.com/nordix/meridio/pkg/nsp/types"
)

type Option func(*KeepAlive)

func WithRegistry(registry types.TargetRegistry) Option {
	return func(ka *KeepAlive) {
		ka.TargetRegistry = registry
	}
}

func WithContextTimeout(ctx context.Context) Option {
	return func(ka *KeepAlive) {
		ka.Timeout = ctx.Done
	}
}

func WithTimeout(delayTime time.Duration) Option {
	return func(ka *KeepAlive) {
		ka.Timeout = func() <-chan struct{} {
			return delay(delayTime)
		}
	}
}

func delay(d time.Duration) <-chan struct{} {
	channel := make(chan struct{}, 1)
	go func() {
		<-time.After(d)
		channel <- struct{}{}
	}()
	return channel
}
