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
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type NameGenerator interface {
	Generate(prefix string, maxLength int) string
	Release(name string)
}

type RandomGenerator struct {
	mu        sync.Mutex
	usedNames map[string]struct{}
}

// TODO: make sure the generated name is within range (probably the prefix length should be limited as well)
func (rg *RandomGenerator) Generate(prefix string, maxLength int) string {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	if rg.usedNames == nil {
		rg.usedNames = make(map[string]struct{})
	}
	randomName := ""
	for randomName == "" {
		rand.Seed(time.Now().UnixNano())
		randomID := rand.Intn(1000)
		randomName = prefix + strconv.Itoa(randomID)
		if _, ok := rg.usedNames[randomName]; ok {
			randomName = ""
		}
	}
	rg.usedNames[randomName] = struct{}{}
	return randomName
}

func (rg *RandomGenerator) Release(name string) {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	delete(rg.usedNames, name)
}

type CounterGenerator struct {
	mu        sync.Mutex
	usedNames map[string]struct{}
}

// TODO: make sure the generated name is within range (probably the prefix length should be limited as well)
func (cg *CounterGenerator) Generate(prefix string, maxLength int) string {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	if cg.usedNames == nil {
		cg.usedNames = make(map[string]struct{})
	}
	selected := 0
	selectedName := fmt.Sprintf("%s%d", prefix, selected)
	length := maxLength - len(prefix)
	length = int(math.Pow(10, float64(length)))
	length -= 1
	for selected < length {
		if _, ok := cg.usedNames[selectedName]; !ok {
			break
		}
		selected++
		selectedName = fmt.Sprintf("%s%d", prefix, selected)
	}
	cg.usedNames[selectedName] = struct{}{}
	return selectedName
}

func (cg *CounterGenerator) Release(name string) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	delete(cg.usedNames, name)
}
