/*
Copyright (c) 2023 Nordix Foundation

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

package bird

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

var regexRouteCount *regexp.Regexp = regexp.MustCompile(`Total:\s+([0-9]+)`)

func ParseRouteCount(input string) (uint64, error) {
	var err error
	var rc uint64 = 0
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		if match := regexRouteCount.FindStringSubmatch(scanner.Text()); match != nil {
			rc, err = strconv.ParseUint(match[1], 10, 32)
			break
		}
	}
	return rc, err
}
