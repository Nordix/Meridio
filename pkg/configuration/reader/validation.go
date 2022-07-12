/*
Copyright (c) 2022 Nordix Foundation

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

package reader

import (
	"regexp"
)

const (
	ByteMatchPattern = `^(sctp|tcp|udp)\[[0-9]+ *: *[124]\]( *& *0x[0-9a-f]+)? *= *([0-9]+|0x[0-9a-f]+)$`
)

func ValidByteMatch(byteMatch string) bool {
	res, _ := regexp.MatchString(ByteMatchPattern, byteMatch)
	return res
}
