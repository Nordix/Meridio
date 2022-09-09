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
	"fmt"
	golog "log"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	nsmlog "github.com/networkservicemesh/sdk/pkg/tools/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger The global logger
var Logger logr.Logger

// FromContextOrGlobal return a logger from the passed context or the
// global logger.
func FromContextOrGlobal(ctx context.Context) logr.Logger {
	if logger, err := logr.FromContext(ctx); err == nil {
		return logger
	}
	return Logger
}

// New returns a new logger. The level may be "DEBUG" (V(1)) or "TRACE" (V(2)),
// any other string (e.g. "") is interpreted as "INFO" (V(0)). On first call
// the global Logger is set.
func New(name, level string) logr.Logger {
	logger := newLogger(level).WithName(name)
	once.Do(func() { Logger = logger })
	return logger
}

var once sync.Once

// Fatal log the message using the passed logger and terminate
func Fatal(logger logr.Logger, msg string, keysAndValues ...interface{}) {
	if z := zapLogger(logger); z != nil {
		z.Sugar().Fatalw(msg, keysAndValues...)
	} else {
		// Fallback to go default
		golog.Fatal(msg, keysAndValues)
	}
}

// NSMLogger return a logger to use for NSM logging.
func NSMLogger(logger logr.Logger) nsmlog.Logger {
	// Get the zap logger
	z := zapLogger(logger)
	if z == nil {
		panic("NSMLogger: Can't get the Zap logger")
	}
	nsmz := z.With(zap.String("subsystem", "NSM"))
	return &nsmLogger{
		z: nsmz,
		s: nsmz.Sugar(),
	}
}

// Called before "main()". Pre-set a global logger
func init() {
	Logger = newLogger("").WithName("Meridio")
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02T15:04:05.999-07:00"))
}

func levelEncoder(lvl zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch lvl {
	case zapcore.InfoLevel:
		enc.AppendString("info")
	case zapcore.WarnLevel:
		enc.AppendString("warning")
	case zapcore.ErrorLevel:
		enc.AppendString("error")
	case zapcore.DPanicLevel:
		enc.AppendString("critical")
	case zapcore.PanicLevel:
		enc.AppendString("critical")
	case zapcore.FatalLevel:
		enc.AppendString("critical")
	default:
		enc.AppendString("debug")
	}
}

func newLogger(level string) logr.Logger {
	var lvl int
	switch level {
	case "DEBUG":
		lvl = -1
	case "TRACE":
		lvl = -2
	default:
		lvl = 0
	}
	zc := zap.NewProductionConfig()
	zc.Level = zap.NewAtomicLevelAt(zapcore.Level(lvl))
	zc.DisableStacktrace = true
	zc.DisableCaller = true
	zc.EncoderConfig.NameKey = "service_id"
	zc.EncoderConfig.LevelKey = "severity"
	zc.EncoderConfig.TimeKey = "timestamp"
	zc.EncoderConfig.MessageKey = "message"
	//zc.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder (almost works)
	zc.EncoderConfig.EncodeTime = timeEncoder
	zc.EncoderConfig.EncodeLevel = levelEncoder
	zc.Encoding = "json"
	z, err := zc.Build()
	if err != nil {
		panic(fmt.Sprintf("Can't create a zap logger (%v)?", err))
	}
	return zapr.NewLogger(z.With(
		zap.String("version", "1.0.0"), zap.Namespace("extra_data")))
}

// zapLogger returns the underlying zap.Logger.
// NOTE; If exported this breaks the use of different log implementations!
func zapLogger(logger logr.Logger) *zap.Logger {
	if underlier, ok := logger.GetSink().(zapr.Underlier); ok {
		return underlier.GetUnderlying()
	} else {
		return nil
	}
}

// NSM logger;

type nsmLogger struct {
	z *zap.Logger
	s *zap.SugaredLogger
}

func (l *nsmLogger) Info(v ...interface{}) {
	l.s.Info(v...)
}
func (l *nsmLogger) Infof(format string, v ...interface{}) {
	l.s.Infof(format, v...)
}
func (l *nsmLogger) Warn(v ...interface{}) {
	l.s.Info(v...)
}
func (l *nsmLogger) Warnf(format string, v ...interface{}) {
	l.s.Infof(format, v...)
}
func (l *nsmLogger) Error(v ...interface{}) {
	l.s.Error(v...)
}
func (l *nsmLogger) Errorf(format string, v ...interface{}) {
	l.s.Errorf(format, v...)
}
func (l *nsmLogger) Fatal(v ...interface{}) {
	l.s.Fatal(v...)
}
func (l *nsmLogger) Fatalf(format string, v ...interface{}) {
	l.s.Fatalf(format, v...)
}
func (l *nsmLogger) Debug(v ...interface{}) {
	l.s.Debug(v...)
}
func (l *nsmLogger) Debugf(format string, v ...interface{}) {
	l.s.Debugf(format, v...)
}
func (l *nsmLogger) Trace(v ...interface{}) {
	if l.z.Core().Enabled(-2) {
		l.s.Debug(v...)
	}
}
func (l *nsmLogger) Tracef(format string, v ...interface{}) {
	if l.z.Core().Enabled(-2) {
		l.s.Debugf(format, v...)
	}
}
func (l *nsmLogger) Object(k, v interface{}) {
	l.z.Info("Object", zap.Any(fmt.Sprintf("%v", k), v))
}
func (l *nsmLogger) WithField(key, value interface{}) nsmlog.Logger {
	z := l.z.With(zap.Any(fmt.Sprintf("%v", key), value))
	return &nsmLogger{
		z: z,
		s: z.Sugar(),
	}
}
