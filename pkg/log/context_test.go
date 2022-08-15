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

package log_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/nordix/meridio/pkg/log"
)

func Test(t *testing.T) {
	type args struct {
		parent context.Context
		logger log.Logger
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "nil context and default logger",
			args: args{
				parent: nil,
				logger: log.NewDefaultLogger(),
			},
		},
		{
			name: "context and logrus logger",
			args: args{
				parent: context.Background(),
				logger: log.NewLogrusLogger(nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContext := log.WithLogger(tt.args.parent, tt.args.logger)
			gotLogger := log.FromContext(gotContext)
			if !reflect.DeepEqual(gotLogger, tt.args.logger) {
				t.Errorf("WithLogger() = %v, want %v", gotLogger, tt.args.logger)
			}
		})
	}
}
