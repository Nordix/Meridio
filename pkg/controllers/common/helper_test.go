/*
Copyright (c) 2025 OpenInfra Foundation Europe

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

package common_test

import (
	"testing"

	"github.com/nordix/meridio/pkg/controllers/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGetDesiredPodSecurityContext(t *testing.T) {
	sysctls := &corev1.PodSecurityContext{
		Sysctls: []corev1.Sysctl{{Name: "net.ipv4.conf.all.forwarding", Value: "1"}},
	}
	fsGroup := &corev1.PodSecurityContext{FSGroup: &[]int64{2000}[0]}

	tests := []struct {
		name    string
		desired *corev1.PodSecurityContext
		current *corev1.PodSecurityContext
		want    *corev1.PodSecurityContext
	}{
		{
			name:    "Desired security context overwrites missing current one",
			desired: sysctls,
			current: nil,
			want:    sysctls,
		},
		{
			name:    "Desired security context overwrites current one",
			desired: sysctls,
			current: fsGroup,
			want:    sysctls,
		},
		{
			name:    "Missing desired and current security context remains nil",
			desired: nil,
			current: nil,
			want:    nil,
		},
		{
			name:    "Empty desired security context does not overwrite empty current one",
			desired: &corev1.PodSecurityContext{},
			current: nil,
			want:    nil,
		},
		{
			name:    "Missing desired security context does not overwrite empty current one",
			desired: nil,
			current: &corev1.PodSecurityContext{},
			want:    &corev1.PodSecurityContext{},
		},
		{
			name:    "Missing desired security context clears current one",
			desired: nil,
			current: fsGroup,
			want:    &corev1.PodSecurityContext{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, common.GetDesiredPodSecurityContext(tt.desired, tt.current))
		})
	}
}
