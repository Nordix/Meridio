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

package log

import (
	"context"
)

const (
	loggerKey contextKeyType = "logger"
)

type contextKeyType string

// WithLogger stores logger in Context
func WithLogger(parent context.Context, logger Logger) context.Context {
	if parent == nil {
		parent = context.WithValue(context.Background(), loggerKey, logger)
	}
	return context.WithValue(parent, loggerKey, logger)
}

// FromContext returns logger from context
func FromContext(ctx context.Context) Logger {
	v, ok := ctx.Value(loggerKey).(Logger)
	if ok {
		return v
	}
	return NewEmptyLogger()
}
