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
	"fmt"
	"io"
	standardLog "log"
	"os"
)

const (
	tracePrefix = "[TRACE]"
	debugPrefix = "[DEBUG]"
	infoPrefix  = "[INFO]"
	warnPrefix  = "[WARN]"
	errorPrefix = "[ERROR]"
	fatalPrefix = "[FATAL]"
)

type logger struct {
	prefix string
	level  Level
	logger *standardLog.Logger
	output io.Writer
}

func NewDefaultLogger() *logger {
	output := os.Stdout
	l := &logger{
		prefix: "",
		level:  InfoLevel,
		logger: standardLog.New(output, "", 0),
		output: output,
	}
	return l
}

func (l *logger) Trace(format string, v ...interface{}) {
	l.print(TraceLevel, format, v...)
}

func (l *logger) Debug(format string, v ...interface{}) {
	l.print(DebugLevel, format, v...)
}

func (l *logger) Info(format string, v ...interface{}) {
	l.print(InfoLevel, format, v...)
}

func (l *logger) Warn(format string, v ...interface{}) {
	l.print(WarnLevel, format, v...)
}

func (l *logger) Error(format string, v ...interface{}) {
	l.print(ErrorLevel, format, v...)
}

func (l *logger) Fatal(format string, v ...interface{}) {
	l.print(FatalLevel, format, v...)
	os.Exit(1)
}

func (l *logger) print(level Level, format string, v ...interface{}) {
	if l.level < level {
		return
	}
	l.logger.Println("", getPrefix(level), l.prefix, fmt.Sprintf(format, v...))
}

func (l *logger) SetOutput(out io.Writer) {
	l.output = out
	l.logger.SetOutput(out)
}

func (l *logger) SetLevel(level Level) {
	l.level = level
}

func (l *logger) WithField(key, value interface{}) Logger {
	prefix := fmt.Sprintf("[%s:%s]", key, value)
	return &logger{
		prefix: prefix,
		level:  l.level,
		logger: standardLog.New(l.output, "", 0),
	}
}

func getPrefix(level Level) string {
	switch level {
	case TraceLevel:
		return tracePrefix
	case DebugLevel:
		return debugPrefix
	case InfoLevel:
		return infoPrefix
	case WarnLevel:
		return warnPrefix
	case ErrorLevel:
		return errorPrefix
	case FatalLevel:
		return fatalPrefix
	}
	return ""
}
