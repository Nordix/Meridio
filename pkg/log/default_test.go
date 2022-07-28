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
	"bytes"
	"strings"
	"testing"

	"github.com/nordix/meridio/pkg/log"
	"github.com/stretchr/testify/assert"
)

type loggerFunc func(log.Logger)

func Test_logger_SetLevel(t *testing.T) {
	tests := []struct {
		name         string
		logging      loggerFunc
		level        log.Level
		numberOfLine int
	}{
		{
			name:         "no log",
			logging:      func(l log.Logger) {},
			numberOfLine: 0,
		},
		{
			name: "trace",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.TraceLevel,
			numberOfLine: 5,
		},
		{
			name: "debug",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.DebugLevel,
			numberOfLine: 4,
		},
		{
			name: "info",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.InfoLevel,
			numberOfLine: 3,
		},
		{
			name: "warn",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.WarnLevel,
			numberOfLine: 2,
		},
		{
			name: "error",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.ErrorLevel,
			numberOfLine: 1,
		},
		{
			name: "fatal",
			logging: func(l log.Logger) {
				l.Trace("a")
				l.Debug("b")
				l.Info("c")
				l.Warn("d")
				l.Error("e")
			},
			level:        log.FatalLevel,
			numberOfLine: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewDefaultLogger()
			logger.SetLevel(tt.level)
			buf := new(bytes.Buffer)
			logger.SetOutput(buf)
			tt.logging(logger)
			logs := logsToArray(buf.String())
			assert.Equal(t, tt.numberOfLine, len(logs))
		})
	}
}

func Test_logger_WithField(t *testing.T) {
	testSubSystemName := "testSubSystem"
	testSubSubSystemName := "testSubSubSystem"

	logger := log.NewDefaultLogger()
	logger.SetLevel(log.DebugLevel)

	buf := new(bytes.Buffer)
	logger.SetOutput(buf)

	testSubsystem := logger.WithField(log.SubSystem, testSubSystemName)

	logger.Info("a")
	testSubsystem.Info("a")

	logs := logsToArray(buf.String())
	assert.Equal(t, 2, len(logs))

	bufSubSystem := new(bytes.Buffer)
	testSubsystem.SetOutput(bufSubSystem)

	logger.Info("b")
	testSubsystem.Info("b")

	logs = logsToArray(buf.String())
	assert.Equal(t, 3, len(logs))

	logs = logsToArray(bufSubSystem.String())
	assert.Equal(t, 1, len(logs))
	for _, log := range logs {
		assert.Contains(t, log, testSubSystemName)
	}

	testSubSubSystem := testSubsystem.WithField(log.SubSystem, testSubSubSystemName)
	bufSubSubSystem := new(bytes.Buffer)
	testSubSubSystem.SetOutput(bufSubSubSystem)

	testSubSubSystem.Info("c")
	testSubSubSystem.Info("c")

	logs = logsToArray(bufSubSubSystem.String())
	assert.Equal(t, 2, len(logs))
	for _, log := range logs {
		assert.Contains(t, log, testSubSubSystemName)
	}
}

func logsToArray(logs string) []string {
	l := strings.Split(strings.ReplaceAll(logs, "\r\n", "\n"), "\n")
	res := []string{}
	for _, log := range l {
		if log == "" {
			continue
		}
		res = append(res, log)
	}
	return res
}
