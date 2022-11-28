// Copyright (c) 2022 Nordix Foundation.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common_test

import (
	"encoding/json"
	"testing"

	"github.com/nordix/meridio/pkg/controllers/common"
	"github.com/stretchr/testify/require"
)

var NetworkAnnotationValidityTests = []struct {
	cfg   string
	valid bool
}{
	{
		`sysctl-tuning@dummy`,
		true,
	},
	{
		`red/sysctl-tuning@dummy`,
		true,
	},
	{
		`red?black/sysctl-tuning@dummy`,
		false,
	},
	{
		`black/sysctl=tuning@dummy`,
		false,
	},
	{
		`sysctl-tuning@dumm:y`,
		false,
	},
	{
		`sysctl-tuning@dum my`,
		false,
	},
	{
		`sysctl-tuning@dum/my`,
		false,
	},
	{
		`sysctl-tuning@aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`,
		false,
	},
}

var NetworkAnnotationTests = []struct {
	cfg, exp string
	match    bool
}{
	{
		`[ { "name" : "sysctl-tuning", "interface": "dummy" } ]`,
		`[{"name":"sysctl-tuning","namespace":"default","interface":"dummy"}]`,
		true,
	},
	{
		`[
			{ "name" : "sysctl-tuning", "interface": "dummy" },
			{ "name" : "sysctl-tuning", "namespace": "red" },
			{ "name" : "mynad" }
		]`,
		`[{"name":"sysctl-tuning","namespace":"default","interface":"dummy"},{"name":"sysctl-tuning","namespace":"red"},{"name":"mynad","namespace":"default"}]`,
		true,
	},
	{
		`[ { "name" : "mynad", "interface": "ext", "ips": "10.0.0.1", "mac": "22:f7:86:12:aa:4f" } ]`,
		`[{"name":"mynad","namespace":"default","interface":"ext"}]`,
		true,
	},
	{
		`sysctl-tuning@dummy`,
		`[{"name":"sysctl-tuning","namespace":"default","interface":"dummy"}]`,
		true,
	},
	{
		`sysctl-tuning@dummy,red/mynad@net1`,
		`[{"name":"sysctl-tuning","namespace":"default","interface":"dummy"},{"name":"mynad","namespace":"red","interface":"net1"}]`,
		true,
	},
}

var NetworkAttachmentMapTests = []struct {
	new, present string
	contains     bool
}{
	{
		`[ { "name" : "sysctl-tuning", "interface": "dummy" } ]`,
		`[ { "name" : "mynad", "namespace" : "default" , "interface":"ext" },{ "name" : "sysctl-tuning" , "interface":"dummy" } ]`,
		true,
	},
	{
		`[ { "name" : "sysctl-tuning", "namespace" : "default", "interface": "dummy" } ]`,
		`[ { "name" : "mynad", "namespace" : "default", "interface": "dummy"} ]`,
		false,
	},
	{
		`[ { "name" : "mynad", "namespace" : "default", "interface": "ext" } ]`,
		`[ { "name" : "mynad", "namespace" : "default"} ]`,
		false,
	},
	{
		`[ { "name" : "mynad", "namespace" : "default", "interface": "ext" } ]`,
		`default/mynad`,
		false,
	},
	{
		`sysctl-tuning@dummy`,
		`[ { "name" : "mynad", "namespace" : "default" , "interface":"ext" },{ "name" : "sysctl-tuning" , "interface":"dummy" } ]`,
		true,
	},
}

func Test_NetworkAnnotationValidity(t *testing.T) {
	for _, nt := range NetworkAnnotationValidityTests {
		// goal is to check validity of the regex patterns in use
		_, err := common.GetNetworkAnnotation(nt.cfg, "default")
		if nt.valid {
			require.NoError(t, err, nt.cfg)
		} else {
			require.Error(t, err, nt.cfg)
		}
	}
}

func Test_NetworkAnnotationMarshal(t *testing.T) {

	for _, nt := range NetworkAnnotationTests {
		// parse network annotation
		networks, err := common.GetNetworkAnnotation(nt.cfg, "default")
		require.NoError(t, err)

		// convert to json format
		enc, err := json.Marshal(networks)
		require.NoError(t, err)
		if nt.match {
			require.Equal(t, string(enc), nt.exp)
		} else {
			require.NotEqual(t, string(enc), nt.exp)
		}
	}
}

func Test_NetworkAttachmentMap(t *testing.T) {

	for _, nt := range NetworkAttachmentMapTests {
		networksPresent, err := common.GetNetworkAnnotation(nt.present, "default")
		require.NoError(t, err)
		networksNew, err := common.GetNetworkAnnotation(nt.new, "default")
		require.NoError(t, err)

		m := common.MakeNetworkAttachmentSpecMap(networksPresent)
		contains := true
		// Check if any of the new networks are already present
		for _, n := range networksNew {
			if _, ok := m[*n]; !ok {
				contains = false
			}
		}
		require.Equal(t, contains, nt.contains)
	}
}
