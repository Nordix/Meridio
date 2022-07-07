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

func Test_RandomGenerator(t *testing.T) {
	generator := &interfacename.RandomGenerator{}

	stringGenerated1 := generator.Generate("", 10)
	assert.NotEmpty(t, stringGenerated1)
	assert.LessOrEqual(t, len(stringGenerated1), 10)

	stringGenerated2 := generator.Generate("abc", 10)
	assert.Contains(t, stringGenerated2, "abc")
	assert.LessOrEqual(t, len(stringGenerated2), 10)

	assert.NotEqual(t, stringGenerated1, stringGenerated2)
}

func Test_CounterGenerator(t *testing.T) {
	generator := &interfacename.CounterGenerator{}

	stringGenerated := generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-0")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-1")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-2")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-3")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-4")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-5")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-6")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-7")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-8")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-9")
	stringGenerated = generator.Generate("nsm-", 5)
	assert.Equal(t, stringGenerated, "nsm-9")
	stringGenerated = generator.Generate("nsm-", 6)
	assert.Equal(t, stringGenerated, "nsm-10")
}
