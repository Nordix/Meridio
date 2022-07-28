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
	"io"
	"os"
)

type emptyLogger struct {
}

func NewEmptyLogger() *emptyLogger {
	el := &emptyLogger{}
	return el
}

func (el *emptyLogger) Trace(format string, v ...interface{}) {
}

func (el *emptyLogger) Debug(format string, v ...interface{}) {
}

func (el *emptyLogger) Info(format string, v ...interface{}) {
}

func (el *emptyLogger) Warn(format string, v ...interface{}) {
}

func (el *emptyLogger) Error(format string, v ...interface{}) {
}

func (el *emptyLogger) Fatal(format string, v ...interface{}) {
	os.Exit(1)
}

func (el *emptyLogger) SetOutput(out io.Writer) {
}

func (el *emptyLogger) SetLevel(level Level) {
}

func (el *emptyLogger) WithField(key, value interface{}) Logger {
	return &emptyLogger{}
}
