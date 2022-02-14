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
	"fmt"
)

// Function definition to be executed and retried.
type RetryableFunc func() error

// Do runs the function passed in parameter (retryableFunc) until the
// retry condition is fulfilled or until the context is Done.
// The following options are supported:
// - Retry condition: condition for retries to terminate (e.g. retryableFunc has not returned any error, max attempts...)
// - Retry trigger: trigger to continue in the next try (e.g. Delay, an system event...)
// - Context: context that can be used to terminate the function
// By default, these options are applied:
// - Retry condition: retryableFunc has not returned any error
// - Retry trigger: 10 milliseconds delay
// - Context: empty context
func Do(retryableFunc RetryableFunc, options ...Option) error {
	config := newConfig()
	for _, opt := range options {
		opt(config)
	}

	var errFinal error

retry:
	for {
		err := retryableFunc()

		if err != nil {
			errFinal = fmt.Errorf("%w; %v", errFinal, err) // todo
		}

		if !config.retryConditionFunc(err) {
			break
		}

		select {
		case <-config.retryTriggerFunc():
		case <-config.context.Done():
			break retry
		}
	}
	return errFinal
}
