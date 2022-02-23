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

package retry

import (
	"context"
	"time"
)

type Option func(*Config)

// WithMaxAttempts specifies the maximum number of attempts/retries
// If no error is returned by the retryable function, then the
// retries will finished before the number of max attempts is reached.
// WithMaxAttempts sets the retry condition function.
func WithMaxAttempts(attempts uint) Option {
	var currentAttempt uint = 0
	return WithRetryCondition(func(err error) bool {
		if currentAttempt >= attempts {
			return false
		}
		currentAttempt++
		return defaultRetryCondition(err)
	})
}

// WithDelay sets a retry trigger with a 10 milliseconds delay
// WithDelay sets the retry trigger function.
func WithDelay(delayTime time.Duration) Option {
	return WithRetryTrigger(func() <-chan struct{} {
		return delay(delayTime)
	})
}

func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.context = ctx
	}
}

// WithRetryCondition sets the retry condition function.
func WithRetryCondition(retryCondition RetryCondition) Option {
	return func(c *Config) {
		c.retryConditionFunc = retryCondition
	}
}

// WithRetryTrigger sets the retry trigger function.
func WithRetryTrigger(retryTrigger RetryTrigger) Option {
	return func(c *Config) {
		c.retryTriggerFunc = retryTrigger
	}
}

// WithErrorIngnored will not stop the loop if no error is returned by
// the retryable function.
// WithErrorIngnored sets the retry condition function.
func WithErrorIngnored() Option {
	return WithRetryCondition(func(err error) bool {
		return true
	})
}
