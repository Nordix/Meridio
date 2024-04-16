/*
Copyright (c) 2024 Nordix Foundation

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

package debug

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
)

const Version = "v0.0.1"

var MeridioVersion = "(unknown)"

type Export struct {
	// Version represents the version of the Export type.
	Version        string `json:"version"`
	MeridioVersion string `json:"meridio-version"`
	// UnixTime represents the time at which the info has been exported.
	UnixTime             int64               `json:"unix-time"`
	NetworkInterfaces    []*NetworkInterface `json:"network-interfaces"`
	Rules                []*Rule             `json:"rules"`
	Routes               []*Route            `json:"route"`
	Neighbors            []*Neighbor         `json:"neighbors"`
	System               *System             `json:"system"`
	EnvironmentVariables []string            `json:"environment-variables"`
	// TODO: netfilter (nftables)
	// TODO: Groups, Users...?
}

type NetworkInterface struct {
	Index       int         `json:"index"`
	Name        string      `json:"name"`
	Mac         string      `json:"mac"`
	IPs         []string    `json:"ips"`
	MTU         int         `json:"mtu"`
	Up          bool        `json:"up"`
	MasterIndex int         `json:"master-index"`
	Statistics  *Statistics `json:"statistics"`
}

type Statistics struct {
	RxPackets uint64 `json:"rx-packets"`
	TxPackets uint64 `json:"tx-packets"`
	RxBytes   uint64 `json:"rx-bytes"`
	TxBytes   uint64 `json:"tx-bytes"`
	RxErrors  uint64 `json:"rx-errors"`
	TxErrors  uint64 `json:"tx-errors"`
	RxDropped uint64 `json:"rx-dropped"`
	TxDropped uint64 `json:"tx-dropped"`
}

type Neighbor struct {
	IP             string `json:"ip"`
	Mac            string `json:"mac"`
	State          string `json:"state"`
	InterfaceIndex int    `json:"interface-index"`
}

type Route struct {
	Table          int      `json:"table"`
	InterfaceIndex int      `json:"interface-index"`
	Destination    string   `json:"destination"`
	Nexthops       []string `json:"nexthops"`
	Gateway        string   `json:"gateway"`
	Source         string   `json:"source"`
}

type Rule struct {
	Table       int    `json:"table"`
	Priority    int    `json:"priority"`
	Mark        int    `json:"mark"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type System struct {
	CPUInfo  []cpu.InfoStat `json:"cpu-info"`
	HostInfo *host.InfoStat `json:"host-info"`
}
