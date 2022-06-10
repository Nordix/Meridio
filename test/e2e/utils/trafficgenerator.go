/*
Copyright (c) 2021-2022 Nordix Foundation

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

type TrafficGeneratorHost struct {
	TrafficGeneratorCommand string
}

type TrafficGenerator interface {
	GetCommand(vip string) string
	AnalyzeTraffic([]byte) (map[string]int, int)
}

func (tgh *TrafficGeneratorHost) SendTraffic(tg TrafficGenerator, trench string, namespace string, vip string) (map[string]int, int) {
	commandString := tgh.TrafficGeneratorCommand
	commandString = strings.ReplaceAll(commandString, "{trench}", trench)
	commandString = strings.ReplaceAll(commandString, "{namespace}", namespace)
	commandString = fmt.Sprintf("%s %s", commandString, tg.GetCommand(vip))
	command := exec.Command("/bin/sh", "-c", commandString)
	var stdout bytes.Buffer
	command.Stdout = &stdout
	_ = command.Run()
	lastingConn, lostConn := tg.AnalyzeTraffic(stdout.Bytes())
	return lastingConn, lostConn
}

type CTraffic struct {
	NConn int
	Rate  int
}

func (ct *CTraffic) GetCommand(vip string) string {
	return fmt.Sprintf("ctraffic -address %s -nconn %d -rate %d -stats all", vip, ct.NConn, ct.Rate)
}

func (ct *CTraffic) AnalyzeTraffic(output []byte) (map[string]int, int) {
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		panic(err)
	}
	lastingConn := map[string]int{}
	lostConn := 0
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
			lostConn++
		}
	}
	return lastingConn, lostConn
}

type MConnect struct {
	NConn int
}

func (mc *MConnect) GetCommand(vip string) string {
	return fmt.Sprintf("mconnect -address %s -nconn %d -timeout 5m -output json", vip, mc.NConn)
}

func (mc *MConnect) AnalyzeTraffic(output []byte) (map[string]int, int) {
	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		panic(err)
	}
	lastingConn := map[string]int{}
	lostConn := 0
	hosts := data["hosts"].(map[string]interface{})
	for name, connections := range hosts {
		lastingConn[name] = int(connections.(float64))
	}
	return lastingConn, lostConn
}
