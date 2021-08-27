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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type lostConnections map[string]int
type lastingConnections map[string]int

func SendTraffic(trafficGeneratorCMD string, trench string, namespace string, vip string, nconn int, rate int) (lastingConnections, lostConnections, error) {
	hostcmd := trafficGeneratorHostCommand(trafficGeneratorCMD, trench)
	ctrafficcmd := ctrafficClientCommand(vip, nconn, rate)
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s %s", hostcmd, ctrafficcmd))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("%w; %s", err, stderr.String())
	}
	lastingConn, lostConn := analyzeTraffic(stdout.Bytes())
	return lastingConn, lostConn, nil
}

func trafficGeneratorHostCommand(trafficGeneratorCMD string, trench string) string {
	return strings.ReplaceAll(trafficGeneratorCMD, "{trench}", trench)
}

func ctrafficClientCommand(vip string, nconn int, rate int) string {
	return fmt.Sprintf("ctraffic -address %s -nconn %d -rate %d -stats all", vip, nconn, rate)
}

func analyzeTraffic(ctrafficOutput []byte) (lastingConnections, lostConnections) {
	var data map[string]interface{}
	if err := json.Unmarshal(ctrafficOutput, &data); err != nil {
		panic(err)
	}
	lastingConn := lastingConnections{}
	lostConn := lostConnections{}
	connStats := data["ConnStats"].([]interface{})
	for _, conn := range connStats {
		connStat := conn.(map[string]interface{})
		if connStat == nil || connStat["Host"] == nil || connStat["Err"] == nil {
			continue
		}
		host := connStat["Host"].(string)
		err := connStat["Err"].(string)
		if host == "" {
			continue
		}
		if err == "" {
			lastingConn[host]++
		} else {
			lostConn[host]++
		}
	}
	return lastingConn, lostConn
}
