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

package next_test

import (
	"reflect"
	"testing"

	"github.com/nordix/meridio/pkg/nsp/next"
)

type fakeNextTargetRegistryServerImpl struct {
	*next.NextTargetRegistryServerImpl
	name string
}

func TestBuildNextTargetRegistryChain(t *testing.T) {
	fakeA := &fakeNextTargetRegistryServerImpl{
		NextTargetRegistryServerImpl: &next.NextTargetRegistryServerImpl{},
		name:                         "a",
	}
	type args struct {
		nextTargetRegistryServers []next.NextTargetRegistryServer
	}
	tests := []struct {
		name string
		args args
		want next.NextTargetRegistryServer
	}{
		{
			name: "empty",
			args: args{},
			want: nil,
		},
		{
			name: "one",
			args: args{
				[]next.NextTargetRegistryServer{
					fakeA,
				},
			},
			want: fakeA,
		},
		{
			name: "multiple",
			args: args{
				[]next.NextTargetRegistryServer{
					fakeA,
					&fakeNextTargetRegistryServerImpl{
						NextTargetRegistryServerImpl: &next.NextTargetRegistryServerImpl{},
						name:                         "b",
					},
					&fakeNextTargetRegistryServerImpl{
						NextTargetRegistryServerImpl: &next.NextTargetRegistryServerImpl{},
						name:                         "c",
					},
				},
			},
			want: fakeA,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := next.BuildNextTargetRegistryChain(tt.args.nextTargetRegistryServers...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildNextTargetRegistryChain() = %v, want %v", got, tt.want)
			}
		})
	}
}
