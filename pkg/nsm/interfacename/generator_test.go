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

package interfacename_test

import (
	"testing"

	"github.com/nordix/meridio/pkg/nsm/interfacename"
	"github.com/stretchr/testify/assert"
)

func Test_Generate(t *testing.T) {
	generator := &interfacename.RandomGenerator{}

	stringGenerated := generator.Generate("", 10)
	assert.NotEmpty(t, stringGenerated)
	assert.LessOrEqual(t, len(stringGenerated), 10)

	stringGenerated = generator.Generate("abc", 10)
	assert.Contains(t, stringGenerated, "abc")
	assert.LessOrEqual(t, len(stringGenerated), 10)
}
