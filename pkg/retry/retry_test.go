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

package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nordix/meridio/pkg/retry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func Test_Do_NoError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		return nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 1, attempts)
}

func Test_Do_Failed(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		return errors.New("")
	}, retry.WithDelay(1*time.Nanosecond), retry.WithMaxAttempts(10))
	assert.NotNil(t, err)
	assert.Equal(t, 11, attempts) // 11: first attempt + retries
}

func Test_Do_SeveralAttempts(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		if attempts == 5 {
			return nil
		}
		return errors.New("")
	}, retry.WithDelay(1*time.Nanosecond), retry.WithMaxAttempts(10))
	assert.NotNil(t, err)
	assert.Equal(t, 5, attempts)
}

func Test_Do_Context(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.TODO())

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		if attempts == 5 {
			cancel()
		}
		return errors.New("")
	}, retry.WithDelay(1*time.Nanosecond), retry.WithContext(ctx))
	assert.NotNil(t, err)
	assert.Equal(t, 5, attempts)
}

func Test_Do_WithRetryCondition(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		if attempts >= 2 {
			return errors.New("abc")
		}
		return errors.New("")
	}, retry.WithRetryCondition(func(err error) bool {
		return err.Error() != "abc"
	}))
	assert.NotNil(t, err)
	assert.Equal(t, 2, attempts)
}

func Test_Do_WithRetryTrigger(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.TODO())

	attempts := 0
	err := retry.Do(func() error {
		attempts++
		if attempts == 2 {
			return nil
		}
		cancel()
		return errors.New("")
	}, retry.WithRetryTrigger(func(context.Context) <-chan struct{} {
		channel := make(chan struct{}, 1)
		go func() {
			<-ctx.Done()
			channel <- struct{}{}
		}()
		return channel
	}))
	assert.NotNil(t, err)
	assert.Equal(t, 2, attempts)
}

func Test_Do_WithErrorIgnored(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.TODO())

	attempts := 0
	_ = retry.Do(func() error {
		attempts++
		if attempts == 2 {
			return nil
		} else if attempts == 3 {
			cancel()
		}
		return errors.New("")
	}, retry.WithErrorIngnored(), retry.WithContext(ctx))
	assert.Equal(t, 3, attempts)
}

func Test_Do_WithDelay_GoRoutineLeak(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.TODO())

	go func() {
		_ = retry.Do(func() error {
			return errors.New("")
		}, retry.WithErrorIngnored(), retry.WithContext(ctx), retry.WithDelay(10*time.Second))
	}()
	cancel()
}
