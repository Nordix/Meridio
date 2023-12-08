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

package prefix

import (
	"context"
	"fmt"
	"net"

	"github.com/nordix/meridio/pkg/ipam/types"
)

// Allocate
// 1. Checks if parent length is superior or equals to length
// 2. Generates a first candidate with parent-IP/length
// 3. Gets childs of parent (with store)
// 4. Checks if there is any collision with childs and candidate (If there is no, go to 6)
// 5. If there is a collision, get next prefix, set as candidate, and got to 4, if candidate = first candidate, returns an error
// 6. Save in with the store
// 7. Gets the childs
// 8. Checks if only 1 collision (the one we just added)
func Allocate(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage) (types.Prefix, error) {
	return AllocateWithBlocklist(ctx, parent, name, length, store, []string{})
}

func AllocateWithBlocklist(ctx context.Context, parent types.Prefix, name string, length int, store types.Storage, blocklist []string) (types.Prefix, error) {
	_, parentIPNet, _ := net.ParseCIDR(parent.GetCidr())
	parentLength, _ := parentIPNet.Mask.Size()
	if length <= parentLength {
		return nil, fmt.Errorf("invalid prefix length requested (should be > to %v): %v", parentLength, length)
	}
	_, currentCandidate, err := net.ParseCIDR(fmt.Sprintf("%s/%d", parentIPNet.IP.String(), length))
	if err != nil {
		return nil, fmt.Errorf("failed to ParseCIDR (%s) while allocating prefix (name: %s ; length: %d ; parent: %s): %w", parentIPNet.IP.String(), name, length, parent.GetName(), err)
	}
	firstCandidate := currentCandidate
	childs, err := store.GetChilds(ctx, parent)
	if err != nil {
		return nil, fmt.Errorf("failed to get childs from store while allocating prefix (name: %s ; length: %d ; parent: %s): %w", name, length, parent.GetName(), err)
	}
	childList := prefixSliceToStringSlice(childs)
	var currentCandidatePrefix types.Prefix
	for {
		childList = append(childList, blocklist...)
		for {
			collisions := CollideWith(currentCandidate.String(), childList)
			if len(collisions) <= 0 {
				break
			}
			// TODO: sort collisions
			_, collisionIPNet, _ := net.ParseCIDR(collisions[0]) // Jump over the collision
			_, currentCandidate, _ = net.ParseCIDR(fmt.Sprintf("%s/%d", LastIP(collisionIPNet).String(), length))
			currentCandidate = NextPrefix(currentCandidate)
			if firstCandidate.String() == currentCandidate.String() || !OverlappingPrefixes(parent.GetCidr(), currentCandidate.String()) { // check if prefix contains candidate
				return nil, fmt.Errorf("no more prefix available")
			}
		}
		currentCandidatePrefix = New(name, currentCandidate.String(), parent)
		err = store.Add(ctx, currentCandidatePrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to add (%s) to store while allocating prefix (name: %s ; length: %d ; parent: %s): %w", currentCandidatePrefix, name, length, parent.GetName(), err)
		}
		childs, err = store.GetChilds(ctx, parent)
		if err != nil {
			return nil, fmt.Errorf("failed to get childs from store while allocating prefix (name: %s ; length: %d ; parent: %s): %w", name, length, parent.GetName(), err)
		}
		childList = prefixSliceToStringSlice(childs)
		collisions := PrefixCollideWith(currentCandidatePrefix, childs)
		if len(collisions) == 1 && currentCandidatePrefix.Equals(collisions[0]) {
			break
		}
		err = store.Delete(ctx, currentCandidatePrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to delete (%s) from store while allocating prefix (name: %s ; length: %d ; parent: %s): %w", currentCandidatePrefix, name, length, parent.GetName(), err)
		}
	}
	return currentCandidatePrefix, nil
}

func prefixSliceToStringSlice(prefixes []types.Prefix) []string {
	list := []string{}
	for _, prefix := range prefixes {
		list = append(list, prefix.GetCidr())
	}
	return list
}

func PrefixCollideWith(prefix types.Prefix, childs []types.Prefix) []types.Prefix {
	collisions := []types.Prefix{}
	for _, childPrefix := range childs {
		if OverlappingPrefixes(childPrefix.GetCidr(), prefix.GetCidr()) {
			collisions = append(collisions, childPrefix)
		}
	}
	return collisions
}
