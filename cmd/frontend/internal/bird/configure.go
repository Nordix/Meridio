/*
Copyright (c) 2021-2022 Nordix Foundation

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
	"fmt"
	"os"
	"regexp"
)

// TODO: move bird config write related methods from service.go here

var regexBGPAuthenticationPassword *regexp.Regexp = regexp.MustCompile(`password\s.*([\r\n]+)`)

type routingConfig struct {
	config string // BIRD configuration
	path   string // config file path to write
}

func NewRoutingConfig(path string) *routingConfig {
	return &routingConfig{
		config: "",
		path:   path,
	}
}

// Append -
// Appends input string to configuration
func (r *routingConfig) Append(in string) {
	r.config += in
}

// Apply -
// Applies the configuration by writing it to the config file specified by path
func (r *routingConfig) Apply() error {
	file, err := os.Create(r.path)
	if err != nil {
		return fmt.Errorf("create %v, err: %v", r.path, err)
	}
	defer file.Close()

	_, err = file.WriteString(r.config)
	if err != nil {
		return fmt.Errorf("write config to %v, err: %v", r.path, err)
	}
	return err
}

// String -
// Ensures that sensitive information is not logged
func (r *routingConfig) String() string {
	return regexBGPAuthenticationPassword.ReplaceAllString(r.config, "password ********;${1}")
}
