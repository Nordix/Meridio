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

package logrus

import (
	"io"

	"github.com/nordix/meridio/pkg/log"
	"github.com/sirupsen/logrus"
)

type logger struct {
	entry *logrus.Entry
}

func New() *logger {
	l := &logger{
		entry: logrus.NewEntry(logrus.StandardLogger()),
	}
	return l
}

func (l *logger) Trace(format string, v ...interface{}) {
	l.entry.Tracef(format, v...)
}

func (l *logger) Debug(format string, v ...interface{}) {
	l.entry.Debugf(format, v...)
}

func (l *logger) Info(format string, v ...interface{}) {
	l.entry.Infof(format, v...)
}

func (l *logger) Warn(format string, v ...interface{}) {
	l.entry.Warnf(format, v...)
}

func (l *logger) Error(format string, v ...interface{}) {
	l.entry.Errorf(format, v...)
}

func (l *logger) Fatal(format string, v ...interface{}) {
	l.entry.Fatalf(format, v...)
}

func (l *logger) SetOutput(out io.Writer) {
	l.entry.Logger.SetOutput(out)
}

func (l *logger) SetLevel(level log.Level) {
	l.entry.Logger.SetLevel(logrus.Level(level))
}

func (l *logger) WithField(key, value interface{}) log.Logger {
	return &logger{
		entry: l.entry.WithField(key.(string), value),
	}
}
