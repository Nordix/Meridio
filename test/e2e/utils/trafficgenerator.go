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

type option func(*tgConfig)

type tgConfig struct {
	timeout string
}

func WithTimeout(timeout string) option {
	return func(c *tgConfig) {
		c.timeout = timeout
	}
}

type TrafficGeneratorHost struct {
	TrafficGeneratorCommand string
}

type TrafficGenerator interface {
	GetCommand(ipPort string, protocol string, options ...option) string
	AnalyzeTraffic([]byte) (map[string]int, int, error)
}

func (tgh *TrafficGeneratorHost) SendTraffic(tg TrafficGenerator, trench string, namespace string, ipPort string, protocol string, options ...option) (map[string]int, int) {
	commandString := tgh.TrafficGeneratorCommand
	commandString = strings.ReplaceAll(commandString, "{trench}", trench)
	commandString = strings.ReplaceAll(commandString, "{namespace}", namespace)
	commandString = fmt.Sprintf("%s %s", commandString, tg.GetCommand(ipPort, protocol, options...))
	command := exec.Command("/bin/sh", "-c", commandString)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	_ = command.Run()
	lastingConn, lostConn, err := tg.AnalyzeTraffic(stdout.Bytes())
	if err != nil {
		fmt.Printf("Error sending/anaylzing traffic: %v - %s", err, stderr.String())
	}
	return lastingConn, lostConn
}

type CTraffic struct {
	NConn int
	Rate  int
}

func (ct *CTraffic) GetCommand(ipPort string, protocol string, options ...option) string {
	return fmt.Sprintf("ctraffic %s -address %s -nconn %d -rate %d -stats all", getProtocolParameter(protocol), ipPort, ct.NConn, ct.Rate)
}

func (ct *CTraffic) AnalyzeTraffic(output []byte) (map[string]int, int, error) {
	var data map[string]interface{}
	lastingConn := map[string]int{}
	if err := json.Unmarshal(output, &data); err != nil {
		return lastingConn, 0, err
	}
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
	return lastingConn, lostConn, nil
}

type MConnect struct {
	NConn int
}

func (mc *MConnect) GetCommand(ipPort string, protocol string, options ...option) string {
	config := &tgConfig{
		timeout: "2m",
	}
	for _, opt := range options {
		opt(config)
	}
	return fmt.Sprintf("mconnect %s -address %s -nconn %d -timeout %s -output json", getProtocolParameter(protocol), ipPort, mc.NConn, config.timeout)
}

func (mc *MConnect) AnalyzeTraffic(output []byte) (map[string]int, int, error) {
	var data map[string]interface{}
	lastingConn := map[string]int{}
	if err := json.Unmarshal(output, &data); err != nil {
		return lastingConn, 0, err
	}
	lostConn := int(data["failed_connects"].(float64))
	hosts := data["hosts"].(map[string]interface{})
	for name, connections := range hosts {
		lastingConn[name] = int(connections.(float64))
	}
	return lastingConn, lostConn, nil
}

func getProtocolParameter(protocol string) string {
	if strings.ToLower(protocol) == "udp" {
		return "-udp"
	}
	return ""
}

func VIPPort(vip string, port int) string {
	return fmt.Sprintf("%s:%d", vip, port)
}
