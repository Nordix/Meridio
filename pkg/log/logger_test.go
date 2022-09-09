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

/*
Note that nothing is actually checked by this test at the moment. It
exercises logging but you must eyeball the printouts to check that
everything looks ok. E.g. no "SHOULD NOT BE SEEN!!!" should be seen.
*/

package log_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
)

func gattherInfo() string {
	fmt.Println("SHOULD NOT BE SEEN!!!")
	return "Some hard-to-get info"
}

func TestLogger(t *testing.T) {
	log.Logger.Info("From the default logger before New")
	logger := log.New("Meridio-test-global", os.Getenv("FOO"))
	log.Logger.Info("From the default logger AFTER New")

	logger.Info("Started", "at", time.Now())
	log.Logger.Info("Printed by the global logger")
	logger.Error(fmt.Errorf("THIS IS THE ERROR OBJ"), "An error!", "a number", 44)
	logger.V(1).Info("INVISIBLE DEBUG message", "info", "Some important info")
	logger.V(2).Info("INVISIBLE TRACE message")

	logger = log.New("Meridio-test", "DEBUG")
	logger.V(1).Info("Visible DEBUG message", "info", "Some important info")
	logger.V(2).Info("INVISIBLE TRACE message")

	// https://github.com/go-logr/logr/issues/149
	if loggerV := logger.V(2); loggerV.Enabled() {
		fmt.Println("SHOULD NOT BE SEEN!!!")
		// Do something expensive.
		loggerV.Info("here's an expensive log message", "info", gattherInfo())
	}

	logger = log.New("Meridio-test", "TRACE")
	logger.V(1).Info("Visible DEBUG message")
	logger.V(2).Info("Visible TRACE message")

	log.Logger.Info("From the default logger")
	log.Logger.V(1).Info(
		"INVISIBLE DEBUG message", "info", "Some important info")
	log.Logger.V(2).Info("INVISIBLE TRACE message")

	//log.Fatal(logger, "Can't read crucial data", "error", fmt.Errorf("Not found"))
}

func TestNSMLogger(t *testing.T) {
	nsmlogger := log.NSMLogger(log.New("NSMLogger-info", ""))
	if nsmlogger == nil {
		return
	}
	nsmlogger.WithField("scope", "x").Info("Hello")
	nsmlogger.Info("one", "two", "three")
	nsmlogger.Infof("%v, %v, %v", "one", "two", "three")
	nsmlogger.Object(44, "Key is an int")
}

type someHandler struct {
	ctx    context.Context
	logger logr.Logger
	Adr    *net.TCPAddr // Capitalized to make it visible!
}

func newHandler(ctx context.Context, addr string) *someHandler {
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "someHandler")
	adr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(logger, "ResolveTCPAddr", "error", err)
	}
	h := &someHandler{
		ctx:    ctx,
		logger: logger.WithValues("instance", adr),
		Adr:    adr,
	}
	h.logger.Info("Created")
	return h
}

func (h *someHandler) connect() error {
	logger := h.logger.WithValues("func", "connect")
	logger.Info("Called")
	return nil
}

func TestPatterns(t *testing.T) {
	logger := log.New("HandlerApp", "")
	ctx := logr.NewContext(context.TODO(), logger)
	h := newHandler(ctx, "[1000::]:80")
	_ = h.connect()
}
