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

package trench_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/ipam/conduit"
	"github.com/nordix/meridio/pkg/ipam/prefix"
	"github.com/nordix/meridio/pkg/ipam/trench"
	"github.com/nordix/meridio/pkg/ipam/trench/mocks"
	"github.com/nordix/meridio/pkg/ipam/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestSetConduits(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	conduitA := &nspAPI.Conduit{
		Name: "Conduit-A",
		Trench: &nspAPI.Trench{
			Name: "Trench-A",
		},
	}
	conduitB := &nspAPI.Conduit{
		Name: "Conduit-B",
		Trench: &nspAPI.Trench{
			Name: "Trench-A",
		},
	}
	conduitC := &nspAPI.Conduit{
		Name: "Conduit-C",
		Trench: &nspAPI.Trench{
			Name: "Trench-A",
		},
	}

	// Add 1 conduit
	tw1 := mocks.NewMockTrenchWatcher(ctrl)
	tw1.EXPECT().AddConduit(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) (types.Conduit, error) {
		assert.Equal(t, conduitA.GetName(), name)
		c := conduit.New(prefix.New("abc", "172.168.0.0/20", nil), nil, &types.PrefixLengths{})
		return c, nil
	})

	// Remove 1 conduit
	tw2 := mocks.NewMockTrenchWatcher(ctrl)
	tw2.EXPECT().RemoveConduit(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) error {
		assert.Equal(t, conduitA.GetName(), name)
		return nil
	})

	// Add 1, Remove 1, Keep 1
	firstAdd := ""
	tw3 := mocks.NewMockTrenchWatcher(ctrl)
	tw3.EXPECT().AddConduit(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) (types.Conduit, error) {
		assert.True(t, name == conduitA.GetName() || name == conduitB.GetName())
		firstAdd = name
		c := conduit.New(prefix.New("abc", "172.168.0.0/20", nil), nil, &types.PrefixLengths{})
		return c, nil
	})
	tw3.EXPECT().AddConduit(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) (types.Conduit, error) {
		assert.True(t, name == conduitA.GetName() || name == conduitB.GetName())
		assert.NotEqual(t, name, firstAdd)
		c := conduit.New(prefix.New("abc", "172.168.0.0/20", nil), nil, &types.PrefixLengths{})
		return c, nil
	})
	tw3.EXPECT().RemoveConduit(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) error {
		assert.Equal(t, conduitC.GetName(), name)
		return nil
	})

	type args struct {
		ctx             context.Context
		tw              trench.TrenchWatcher
		currentConduits map[string]struct{}
		newConduits     []*nspAPI.Conduit
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "nil Trench Watcher",
			args: args{
				ctx:             context.Background(),
				tw:              nil,
				currentConduits: map[string]struct{}{},
				newConduits:     []*nspAPI.Conduit{},
			},
		},
		{
			name: "Add 1 conduit",
			args: args{
				ctx:             context.Background(),
				tw:              tw1,
				currentConduits: map[string]struct{}{},
				newConduits:     []*nspAPI.Conduit{conduitA},
			},
		},
		{
			name: "Remove 1 conduit",
			args: args{
				ctx: context.Background(),
				tw:  tw2,
				currentConduits: map[string]struct{}{
					conduitA.GetName(): {},
				},
				newConduits: []*nspAPI.Conduit{},
			},
		},
		{
			name: "Add 1, Remove 1, Keep 1",
			args: args{
				ctx: context.Background(),
				tw:  tw3,
				currentConduits: map[string]struct{}{
					conduitA.GetName(): {},
					conduitC.GetName(): {},
				},
				newConduits: []*nspAPI.Conduit{conduitA, conduitB},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trench.SetConduits(tt.args.ctx, tt.args.tw, tt.args.currentConduits, tt.args.newConduits)
		})
	}
}
