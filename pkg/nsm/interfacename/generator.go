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

package interfacename

import (
	"math/rand"
	"strconv"
)

type NameGenerator interface {
	Generate(prefix string, maxLength int) string
}

type RandomGenerator struct {
}

func (rg *RandomGenerator) Generate(prefix string, maxLength int) string {
	randomID := rand.Intn(1000)
	randomName := prefix + strconv.Itoa(randomID)
	return randomName
}