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
package networking

import (
	"crypto/rand"
	"fmt"
	"net"
)

// https://stackoverflow.com/questions/21018729/generate-mac-address-in-go
func GenerateMacAddress() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random set of bytes while generating a mac address: %w", err)
	}

	buf[0] = (buf[0] | 2) & 0xfe // Set local bit, ensure unicast addres

	return buf, nil
}
