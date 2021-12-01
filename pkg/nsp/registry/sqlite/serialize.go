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

package sqlite

import (
	"encoding/json"
	"strings"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
)

const (
	separator = ";"
)

func SerializeIPs(ips []string) string {
	return strings.Join(ips, separator)
}

func DeserializeIPs(ips string) []string {
	return strings.Split(ips, separator)
}

func SerializeContext(context map[string]string) string {
	json, err := json.Marshal(context)
	if err != nil {
		return ""
	}
	return string(json)
}

func DeserializeContext(context string) map[string]string {
	var ctx map[string]string
	err := json.Unmarshal([]byte(context), &ctx)
	if err != nil {
		return map[string]string{}
	}
	return ctx
}

func SerializeStatus(status nspAPI.Target_Status) int32 {
	return int32(status)
}

func DeserializeStatus(status int32) nspAPI.Target_Status {
	return nspAPI.Target_Status(status)
}

func SerializeType(t nspAPI.Target_Type) int32 {
	return int32(t)
}

func DeserializeType(t int32) nspAPI.Target_Type {
	return nspAPI.Target_Type(t)
}
