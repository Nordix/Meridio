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

	"github.com/sirupsen/logrus"
)

type logrusLogger struct {
	entry *logrus.Entry
}

// NewLogrusLogger return a new logger with a logrus backend.
// If the logger parameter is not-nil it will be used as base. This
// allows for instance passing of a base logger with JSON formatter.
func NewLogrusLogger(l *logrus.Logger) *logrusLogger {
	if l != nil {
		return &logrusLogger{
			entry: logrus.NewEntry(l),
		}
	}
	return &logrusLogger{
		entry: logrus.NewEntry(logrus.StandardLogger()),
	}
}

func (l *logrusLogger) Trace(format string, v ...interface{}) {
	l.entry.Tracef(format, v...)
}

func (l *logrusLogger) Debug(format string, v ...interface{}) {
	l.entry.Debugf(format, v...)
}

func (l *logrusLogger) Info(format string, v ...interface{}) {
	l.entry.Infof(format, v...)
}

func (l *logrusLogger) Warn(format string, v ...interface{}) {
	l.entry.Warnf(format, v...)
}

func (l *logrusLogger) Error(format string, v ...interface{}) {
	l.entry.Errorf(format, v...)
}

func (l *logrusLogger) Fatal(format string, v ...interface{}) {
	l.entry.Fatalf(format, v...)
}

func (l *logrusLogger) SetOutput(out io.Writer) {
	l.entry.Logger.SetOutput(out)
}

func (l *logrusLogger) SetLevel(level Level) {
	l.entry.Logger.SetLevel(logrus.Level(level))
}

func (l *logrusLogger) WithField(key, value interface{}) Logger {
	return &logrusLogger{
		entry: l.entry.WithField(key.(string), value),
	}
}
