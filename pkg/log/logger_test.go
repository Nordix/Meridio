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
	"testing"

	"github.com/nordix/meridio/pkg/log"
)

func TestParseLevel(t *testing.T) {
	type args struct {
		level string
	}
	tests := []struct {
		name    string
		args    args
		want    log.Level
		wantErr bool
	}{
		{
			name: "fatal",
			args: args{
				level: "fatal",
			},
			want:    log.FatalLevel,
			wantErr: false,
		},
		{
			name: "fatal uppercase",
			args: args{
				level: "FATAL",
			},
			want:    log.FatalLevel,
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				level: "error",
			},
			want:    log.ErrorLevel,
			wantErr: false,
		},
		{
			name: "error uppercase",
			args: args{
				level: "ERROR",
			},
			want:    log.ErrorLevel,
			wantErr: false,
		},
		{
			name: "warn",
			args: args{
				level: "warn",
			},
			want:    log.WarnLevel,
			wantErr: false,
		},
		{
			name: "warn uppercase",
			args: args{
				level: "WARN",
			},
			want:    log.WarnLevel,
			wantErr: false,
		},
		{
			name: "warning",
			args: args{
				level: "warning",
			},
			want:    log.WarnLevel,
			wantErr: false,
		},
		{
			name: "warning uppercase",
			args: args{
				level: "WARNING",
			},
			want:    log.WarnLevel,
			wantErr: false,
		},
		{
			name: "info",
			args: args{
				level: "info",
			},
			want:    log.InfoLevel,
			wantErr: false,
		},
		{
			name: "info uppercase",
			args: args{
				level: "INFO",
			},
			want:    log.InfoLevel,
			wantErr: false,
		},
		{
			name: "debug",
			args: args{
				level: "debug",
			},
			want:    log.DebugLevel,
			wantErr: false,
		},
		{
			name: "debug uppercase",
			args: args{
				level: "DEBUG",
			},
			want:    log.DebugLevel,
			wantErr: false,
		},
		{
			name: "trace",
			args: args{
				level: "trace",
			},
			want:    log.TraceLevel,
			wantErr: false,
		},
		{
			name: "trace uppercase",
			args: args{
				level: "TRACE",
			},
			want:    log.TraceLevel,
			wantErr: false,
		},
		{
			name: "invalid log level",
			args: args{
				level: "test",
			},
			want:    log.InfoLevel,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := log.ParseLevel(tt.args.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLevel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
