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

package utils_test

import (
	"testing"

	"github.com/nordix/meridio/cmd/frontend/internal/utils"
	"gotest.tools/assert"
)

func Test(t *testing.T) {
	assert.Assert(t, utils.IsIPv4("10.0.0.1"))
	assert.Assert(t, utils.IsIPv4("192.168.0.1"))

	assert.Assert(t, utils.IsIPv6("1000::1"))
	assert.Assert(t, utils.IsIPv6("fe80::1234"))

	assert.Assert(t, !utils.IsIPv4("1000::1"))
	assert.Assert(t, !utils.IsIPv6("192.168.0.1"))

	assert.Assert(t, !utils.IsIPv4("::FFFF:1.2.3.4"))
	assert.Assert(t, utils.IsIPv6("::FFFF:127.0.0.1"))

	ip1 := utils.StrToIPNet("10.0.0.1/20")
	assert.Assert(t, ip1 != nil)
	assert.Equal(t, "10.0.0.1/20", ip1.String())

	ip2 := utils.StrToIPNet("100:0:1::1/64")
	assert.Assert(t, ip2 != nil)
	assert.Equal(t, "100:0:1::1/64", ip2.String())
}
