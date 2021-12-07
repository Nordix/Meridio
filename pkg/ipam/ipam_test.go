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

package ipam_test

import (
	"context"
	"testing"

	"github.com/nordix/meridio/pkg/ipam"
	"github.com/stretchr/testify/assert"
)

func Test_IPv4_AllocateIP(t *testing.T) {
	im := ipam.New()
	assert.NotNil(t, im)

	ip, err := im.AllocateIP(context.TODO(), "169.16.0.0/24")
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.1/24", ip)

	ip, err = im.AllocateIP(context.TODO(), "169.16.0.0/24")
	assert.Nil(t, err)
	assert.Equal(t, "169.16.0.2/24", ip)
}
