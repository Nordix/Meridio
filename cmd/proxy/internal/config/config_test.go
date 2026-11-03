/*
Copyright (c) 2026 OpenInfra Foundation Europe

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

package config

import (
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

// TestTeardownWaitTimeoutDefault pins the production default: with no env override,
// envconfig (same call the proxy binary makes: Process("nsm", ...)) must resolve
// TeardownWaitTimeout to 2s. This is the behavioural contract that the production
// re-add path is block-with-timeout, not skip-immediately (which is the 0 value).
// If someone removes/changes the struct-tag default, this fails.
func TestTeardownWaitTimeoutDefault(t *testing.T) {
	var cfg Config
	require.NoError(t, envconfig.Process("nsm", &cfg))
	require.Equal(t, 2*time.Second, cfg.TeardownWaitTimeout,
		"production default must be a bounded wait (2s), not skip-immediately (0)")
}

// TestTeardownWaitTimeoutEnvOverride confirms the documented env override and the
// disable sentinel (0) are both honoured.
func TestTeardownWaitTimeoutEnvOverride(t *testing.T) {
	t.Setenv("NSM_TEARDOWN_WAIT_TIMEOUT", "0")
	var cfg Config
	require.NoError(t, envconfig.Process("nsm", &cfg))
	require.Equal(t, time.Duration(0), cfg.TeardownWaitTimeout,
		"0 must be accepted as the documented skip-immediately (disable wait) value")
}
