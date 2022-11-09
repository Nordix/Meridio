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

package common_test

import (
	"reflect"
	"testing"

	"github.com/nordix/meridio/pkg/controllers/common"
	corev1 "k8s.io/api/core/v1"
)

func TestCompileEnvironmentVariables(t *testing.T) {
	type args struct {
		allEnv      []corev1.EnvVar
		operatorEnv map[string]string
	}
	tests := []struct {
		name string
		args args
		want []corev1.EnvVar
	}{
		{
			name: "Add non existing env variable",
			args: args{
				allEnv: []corev1.EnvVar{},
				operatorEnv: map[string]string{
					"TEST": "A",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "TEST",
					Value: "A",
				},
			},
		},
		{
			name: "Overwrite empty env variable",
			args: args{
				allEnv: []corev1.EnvVar{
					{
						Name:  "TEST",
						Value: "",
					},
				},
				operatorEnv: map[string]string{
					"TEST": "A",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "TEST",
					Value: "A",
				},
			},
		},
		{
			name: "Ignore non empty env variable",
			args: args{
				allEnv: []corev1.EnvVar{
					{
						Name:  "TEST",
						Value: "B",
					},
				},
				operatorEnv: map[string]string{
					"TEST": "A",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "TEST",
					Value: "B",
				},
			},
		},
		{
			name: "Ignore unknown env variables",
			args: args{
				allEnv: []corev1.EnvVar{
					{
						Name:  "TEST",
						Value: "",
					},
					{
						Name:  "UNKNOWN_A",
						Value: "1",
					},
					{
						Name:  "UNKNOWN_B",
						Value: "2",
					},
				},
				operatorEnv: map[string]string{
					"TEST": "A",
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "TEST",
					Value: "A",
				},
				{
					Name:  "UNKNOWN_A",
					Value: "1",
				},
				{
					Name:  "UNKNOWN_B",
					Value: "2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := common.CompileEnvironmentVariables(tt.args.allEnv, tt.args.operatorEnv); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CompileEnvironmentVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}
