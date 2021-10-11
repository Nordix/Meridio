/*
Copyright (c) 2021 Nordix Foundation

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
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// LookupCli -
// Looks up birdc path
func LookupCli() (string, error) {
	path, err := exec.LookPath("birdc")
	if err != nil {
		err = errors.New("Birdc not found, err: " + err.Error())
	}
	return path, err
}

// CliCmd -
// Executes birdc commands
func CliCmd(ctx context.Context, lp string, arg ...string) (string, error) {
	if lp == "" {
		path, err := LookupCli()
		if err != nil {
			return path, err
		}
		lp = path
	}

	cmd := exec.CommandContext(ctx, lp, arg...)
	stdoutStderr, err := cmd.CombinedOutput()
	return string(stdoutStderr), err
}

// CheckCli -
// Checks if birdc is available
func CheckCli(ctx context.Context, lp string) error {
	if lp == "" {
		path, err := LookupCli()
		if err != nil {
			return err
		}
		lp = path
	}

	cmd := exec.CommandContext(ctx, lp, "show", "status")
	stdoutStderr, err := cmd.CombinedOutput()
	stringOut := string(stdoutStderr)
	if err != nil {
		return errors.New(cmd.String() + ": " + err.Error() + ": " + stringOut)
	}
	return nil
}

// Run -
// Starts BIRD process with the config file (blocks)
// Based on monitorLogs settings stderr of the started BIRD process can be monitored,
// so that important log snippets get appended to the container's log
func Run(ctx context.Context, configFile string, monitorLogs bool) error {
	if !monitorLogs {
		if stdoutStderr, err := exec.CommandContext(ctx, "bird", "-d", "-c", configFile).CombinedOutput(); err != nil {
			logrus.Errorf("BIRD Start: err: \"%v\", out: %s", err, stdoutStderr)
			return err
		}
	} else {
		cmd := exec.CommandContext(ctx, "bird", "-d", "-c", configFile)
		// get stderr pipe reader that will be connected with the process' stderr by Start()
		pipe, err := cmd.StderrPipe()
		if err != nil {
			logrus.Errorf("BIRD Start: stderr pipe err: \"%v\"", err)
			return err
		}

		// Note: Probably not needed at all, as due to the use of CommandContext()
		// Start() would kill the process as soon context becomes done. Which should
		// lead to an EOF on stderr anyways.
		go func() {
			// make sure bufio Scan() can be breaked out from when context is done
			w, ok := cmd.Stderr.(*os.File)
			if !ok {
				// not considered a deal-breaker at the moment; see above note
				logrus.Debugf("BIRD Start: cmd.Stderr not *os.File")
				return
			}
			// when context is done, close File thus signalling EOF to bufio Scan()
			defer w.Close()
			<-ctx.Done()
			logrus.Infof("BIRD Start: context closed, terminate log monitoring...")
		}()

		// start the process (BIRD)
		if err := cmd.Start(); err != nil {
			logrus.Errorf("BIRD Start: start err: \"%v\"", err)
			return err
		}
		if err := monitorOutput(pipe); err != nil {
			logrus.Errorf("BIRD Start: scanner err: \"%v\"", err)
			return err
		}
		// wait until process concludes
		// (should only get here after stderr got closed or scanner returned error)
		if err := cmd.Wait(); err != nil {
			logrus.Errorf("BIRD Start: err: \"%v\"", err)
			return err
		}
	}
	return nil
}

var regexWarn *regexp.Regexp = regexp.MustCompile(`Error|<ERROR>|<BUG>|<FATAL>|<WARNING>`)
var regexInfo *regexp.Regexp = regexp.MustCompile(`<INFO>|BGP session|Connected|Received:|Started|Neighbor|Startup delayed`)

// monitorOutput -
// Keeps reading the output of a BIRD process and adds important
// log snippets to the containers log for debugging purposes
func monitorOutput(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if ok := regexWarn.MatchString(scanner.Text()); ok {
			logrus.Warnf("[bird] %v", scanner.Text())
		} else if ok := regexInfo.MatchString(scanner.Text()); ok {
			logrus.Infof("[bird] %v", scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Configure -
// Orders BIRD via birdc to (try and) apply the config
func Configure(ctx context.Context, lp, configFile string) (string, error) {
	arg := `"` + configFile + `"`
	stringOut, err := CliCmd(ctx, lp, "configure", arg)

	if err != nil {
		return stringOut, err
	} else if !strings.Contains(stringOut, ReconfigInProgress) && !strings.Contains(stringOut, Reconfigured) {
		return stringOut, errors.New("Reconfiguration failed")
	} else {
		return stringOut, nil
	}
}

// Verify -
// Verifies content of config file via birdc
func Verify(ctx context.Context, lp, configFile string) (string, error) {
	arg := `"` + configFile + `"`
	stringOut, err := CliCmd(ctx, lp, "configure", "check", arg)
	if err != nil {
		return stringOut, err
	} else if !strings.Contains(stringOut, ConfigurationOk) {
		return stringOut, errors.New("Verification failed")
	} else {
		logrus.Debugf("VerifyConfig: %v", stringOut)
		return stringOut, nil
	}
}

// ShowProtocolSessions -
// Retrieves detailed routing protocol information via birdc
func ShowProtocolSessions(ctx context.Context, lp, pattern string) (string, error) {
	args := []string{
		"show",
		"protocols",
		"all",
	}
	if pattern != "" {
		args = append(args, `"`+pattern+`"`)
	}
	return CliCmd(ctx, lp, args...)
}

// ShowProtocolSessions -
// Retrieves information on the available BFD sessions (for the given BFD protocol name if any)
func ShowBfdSessions(ctx context.Context, lp, name string) (string, error) {
	args := []string{
		"show",
		"bfd",
		"session",
	}
	if name != "" {
		args = append(args, `'`+name+`'`)
	}
	return CliCmd(ctx, lp, args...)
}
