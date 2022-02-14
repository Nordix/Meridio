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

// RetryTrigger is the function definition specifying when the Do
// function should continue in an new attempt.
type RetryTrigger func() <-chan struct{}

// RetryCondition is the function definition specifying if the Do
// function should continue in an new attempt or not.
type RetryCondition func(error) bool

type Config struct {
	context            context.Context
	retryTriggerFunc   RetryTrigger
	retryConditionFunc RetryCondition
}

func newConfig() *Config {
	return &Config{
		context: context.Background(),
		retryTriggerFunc: func() <-chan struct{} {
			return delay(10 * time.Millisecond)
		},
		retryConditionFunc: defaultRetryCondition,
	}
}

func defaultRetryCondition(err error) bool {
	return err != nil
}

func delay(d time.Duration) <-chan struct{} {
	channel := make(chan struct{}, 1)
	go func() {
		<-time.After(d)
		channel <- struct{}{}
	}()
	return channel
}
