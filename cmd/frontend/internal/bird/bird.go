/*
Copyright (c) 2021-2023 Nordix Foundation

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

package bird

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/nordix/meridio/pkg/log"
)

var regexError *regexp.Regexp = regexp.MustCompile(`Error|<ERROR>|<BUG>|<FATAL>|syntax error`)
var regexInfo *regexp.Regexp = regexp.MustCompile(`<INFO>|<WARNING>|BGP session|Connected|Received:|Started|Neighbor|Startup delayed`)

func NewRoutingService(ctx context.Context, commSocket string, configFile string) *RoutingService {
	return &RoutingService{
		communicationSocket: commSocket,
		configFile:          configFile,
		logger:              log.FromContextOrGlobal(ctx).WithValues("class", "RoutingService", "instance", "BIRD"),
	}
}

type RoutingService struct {
	communicationSocket string // filename (with path) to communicate with birdc
	configFile          string // configuration file (with path)
	logger              logr.Logger
}

// LookupCli -
// Looks up birdc path
func (b *RoutingService) LookupCli() (string, error) {
	path, err := exec.LookPath("birdc")
	if err != nil {
		err = fmt.Errorf("birdc not found: %v", err.Error())
	}
	return path, err
}

// CliCmd -
// Executes birdc commands
func (b *RoutingService) CliCmd(ctx context.Context, lp string, arg ...string) (string, error) {
	if lp == "" {
		path, err := b.LookupCli()
		if err != nil {
			return path, err
		}
		lp = path
	}

	arg = append([]string{"-s", b.communicationSocket}, arg...)
	cmd := exec.CommandContext(ctx, lp, arg...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

// CheckCli -
// Checks if birdc is available
func (b *RoutingService) CheckCli(ctx context.Context, lp string) error {
	if lp == "" {
		path, err := b.LookupCli()
		if err != nil {
			return err
		}
		lp = path
	}

	cmd := exec.CommandContext(ctx, lp, "-s", b.communicationSocket, "show", "status")
	stdoutStderr, err := cmd.CombinedOutput()
	stringOut := string(stdoutStderr)
	if err != nil {
		return fmt.Errorf("%v; %v; %v", cmd.String(), err.Error(), stringOut)
	}
	return nil
}

// Run -
// Starts BIRD process with the config file (blocks)
// Based on monitorLogs settings stderr of the started BIRD process can be monitored,
// in order to append important log snippets to the container's log
func (b *RoutingService) Run(ctx context.Context, monitorLogs bool) error {
	if !monitorLogs {
		if stdoutStderr, err := exec.CommandContext(ctx, "bird", "-d", "-c", b.configFile, "-s", b.communicationSocket).CombinedOutput(); err != nil {
			return fmt.Errorf("error starting/running BIRD: %v, out: %s", err, stdoutStderr)
		}
	} else {
		cmd := exec.CommandContext(ctx, "bird", "-d", "-c", b.configFile, "-s", b.communicationSocket)
		// get stderr pipe reader that will be connected with the process' stderr by Start()
		pipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("error creating stderr pipe for BIRD: %v", err)
		}

		// Note: Probably not needed at all, as due to the use of CommandContext()
		// Start() would kill the process as soon context becomes done. Which should
		// lead to an EOF on stderr anyways.
		go func() {
			// make sure bufio Scan() can be breaked out from when context is done
			w, ok := cmd.Stderr.(*os.File)
			if !ok {
				// not considered a deal-breaker at the moment; see above note
				b.logger.V(1).Info("cmd.Stderr not *os.File")
				return
			}
			// when context is done, close File thus signalling EOF to bufio Scan()
			defer w.Close()
			<-ctx.Done()
			b.logger.Info("Context closed, terminate log monitoring")
		}()

		// start the process (BIRD)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error starting BIRD: %v", err)
		}
		if err := b.monitorOutput(ctx, pipe); err != nil {
			return fmt.Errorf("error during BIRD output monitoring: %v", err)
		}
		// wait until process concludes
		// (should only get here after stderr got closed or scanner returned error)
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("error running BIRD: %v", err)
		}
	}
	return nil
}

// ShutDown -
// Shuts BIRD down via birdc
func (b *RoutingService) ShutDown(ctx context.Context, lp string) error {
	out, err := b.CliCmd(ctx, lp, "down")
	if err != nil {
		err = fmt.Errorf("%v - %v", err, out)
	}
	return err
}

// monitorOutput -
// Keeps reading the output of a BIRD process and adds important
// log snippets to the containers log for debugging purposes
func (b *RoutingService) monitorOutput(ctx context.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ok := regexError.MatchString(scanner.Text()); ok {
			b.logger.Error(fmt.Errorf("BIRD error log"), "monitor output", "out", scanner.Text())
		} else if ok := regexInfo.MatchString(scanner.Text()); ok {
			b.logger.Info("monitor output", "birdc-command-output", scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed on reading the output of bird process: %w", err)
	}
	return nil
}

// Configure -
// Orders BIRD via birdc to (try and) apply the config
func (b *RoutingService) Configure(ctx context.Context, lp string) (string, error) {
	arg := `"` + b.configFile + `"`
	stringOut, err := b.CliCmd(ctx, lp, "configure", arg)

	if err != nil {
		return stringOut, err
	} else if !strings.Contains(stringOut, ReconfigInProgress) && !strings.Contains(stringOut, Reconfigured) {
		return stringOut, errors.New("reconfiguration failed")
	} else {
		return stringOut, nil
	}
}

// Verify -
// Verifies content of config file via birdc
func (b *RoutingService) Verify(ctx context.Context, lp string) (string, error) {
	arg := `"` + b.configFile + `"`
	stringOut, err := b.CliCmd(ctx, lp, "configure", "check", arg)
	if err != nil {
		return stringOut, err
	} else if !strings.Contains(stringOut, ConfigurationOk) {
		return stringOut, errors.New("verification failed")
	} else {
		return stringOut, nil
	}
}

// ShowProtocolSessions -
// Retrieves detailed routing protocol information via birdc
func (b *RoutingService) ShowProtocolSessions(ctx context.Context, lp, pattern string) (string, error) {
	args := []string{
		"show",
		"protocols",
		"all",
	}
	if pattern != "" {
		args = append(args, `"`+pattern+`"`)
	}
	return b.CliCmd(ctx, lp, args...)
}

// ShowProtocolSessions -
// Retrieves information on the available BFD sessions (for the given BFD protocol name if any)
func (b *RoutingService) ShowBfdSessions(ctx context.Context, lp, name string) (string, error) {
	args := []string{
		"show",
		"bfd",
		"session",
	}
	if name != "" {
		args = append(args, `'`+name+`'`)
	}
	return b.CliCmd(ctx, lp, args...)
}

// ShowRouteCount -
// Retrieves number of routes from default BIRD routing tables (master4, master6)
// Note: using filters significantly increases CPU usage
func (b *RoutingService) ShowRouteCount(ctx context.Context, lp string) (string, error) {
	args := []string{
		"show",
		"route",
		"count",
	}
	return b.CliCmd(ctx, lp, args...)
}

// ShowMemory -
// Retrieves memory usage information
func (b *RoutingService) ShowMemory(ctx context.Context, lp string) (string, error) {
	args := []string{
		"show",
		"memory",
	}
	return b.CliCmd(ctx, lp, args...)
}
