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

package nfqlb

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/sirupsen/logrus"
)

type nfqlbFactory struct {
	nfqueue string
}

func NewLbFactory(options ...Option) *nfqlbFactory {
	opts := &nfoptions{
		nfqueue: NFQueues,
	}
	for _, opt := range options {
		opt(opts)
	}

	return &nfqlbFactory{
		nfqueue: opts.nfqueue,
	}
}

// Start -
// Starts nfqlb process in 'flowlb' mode supporting multiple shared mem lbs at once
// https://github.com/Nordix/nfqueue-loadbalancer/blob/98ae93f9137ecc383c61a8bb1a850319bcdfbbb6/src/nfqlb/cmdFlowLb.c#L176
// (Returned context gets cancelled when nfqlb process stops for whatever reason)
//
// Note:
// nfqlb process is supposed to run while the load-balancer container
// is alive and vice versa, thus there's no need for a Stop() function
//
// TODO:
// Consider using the fragment tracking feature of nfqlb (requires a tun dev),
// instead of relying on linux kernel's defragmentation hook for packets coming
// from outside the cluster.
func (nf *nfqlbFactory) Start(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		logrus.Infof("Starting nfqlb process")
		defer cancel()
		cmd := exec.CommandContext(
			ctx,
			nfqlbCmd,
			"flowlb",
			// "--lbshm=",
			// "--mtu=",
			// "--tun=",
			// "--reassembler=",
			"--promiscuous_ping", // accept ICMP Echo (ping) by default
			// "--notargets_fwmark=",
			// "--nolb_fwmark=",
			fmt.Sprintf("--queue=%s", nf.nfqueue),
			fmt.Sprintf("--qlength=%d", qlength),
			// "--ft_shm=",
			// "--ft_size=",
			// "--ft_buckets=",
			// "--ft_frag=",
			// "--ft_ttl=",
		)

		logrus.Debugf("%v", cmd.String())
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Errorf("nfqlb terminated err: \"%v\", out: %s", err, stdoutStderr)
		}
	}()

	return ctx
}

// New -
// Creates new nfqlb shared memory LB
func (nf *nfqlbFactory) New(name string, m int, n int) (types.NFQueueLoadBalancer, error) {
	return NewLb(WithLbName(nf.getTargetSHM(name)), WithMaglevM(m), WithMaglevN(n))
}

func (nf *nfqlbFactory) getTargetSHM(name string) string {
	return fmt.Sprintf("tshm-%s", name)
}

//---------------------------------------------------------

type nfqlb struct {
	name string
	m    int
	n    int
}

// NewLb -
// Creates LB that implements a Stream in nfqlb as a shared mem lb
func NewLb(options ...LbOption) (*nfqlb, error) {
	opts := &lbOptions{
		name: "lb",
	}
	for _, opt := range options {
		opt(opts)
	}

	return &nfqlb{
		name: opts.name,
		m:    opts.m,
		n:    opts.n,
	}, nil
}

// Start -
// Start adds the shared mem lb to nfqlb running in 'flowlb' mode
func (n *nfqlb) Start() error {
	ctx := context.TODO()
	cmd := exec.CommandContext(
		ctx,
		nfqlbCmd,
		"init",
		fmt.Sprintf("--ownfw=%d", ownfw),
		fmt.Sprintf("--shm=%s", n.name),
		fmt.Sprintf("--M=%d", n.m),
		fmt.Sprintf("--N=%d", n.n),
	)

	logrus.Infof("Start nfqlb shared mem lb: %v", cmd.String())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// Delete -
// Delete decreases reference count of the file backing the shared mem lb
//
// Notes:
// - The file is not removed by the OS unless no other references are held
// - Flow rule also store references towards the shared mem lb it's associated with.
//   So they must be removed as well to get rid of a shared mem lb. (But that is
//   the responsiblility of the "user" i.e. the Stream construct.)
// - Previously activated Targets are not deactivated, as that information shall
//  disappear once the shared mem is destroyed.
func (n *nfqlb) Delete() error {
	ctx := context.TODO()
	// unlink the shared mem file
	cmd := exec.CommandContext(
		ctx,
		nfqlbCmd,
		"delete",
		fmt.Sprintf("--shm=%s", n.name),
	)

	logrus.Infof("Delete nfqlb shared mem lb: %v", cmd.String())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// Activate -
// Activate adds a Target with the very identifier to the associated shared mem lb
func (n *nfqlb) Activate(identifier int) error {
	ctx := context.TODO()
	stdoutStderr, err := exec.CommandContext(
		ctx,
		nfqlbCmd,
		"activate",
		fmt.Sprintf("--index=%d", identifier-1),
		fmt.Sprintf("--shm=%s", n.name),
		strconv.Itoa(identifier),
	).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// Deactivate -
// Deactivate removes a Target with the very identifier from associated shared mem lb
func (n *nfqlb) Deactivate(identifier int) error {
	ctx := context.TODO()
	stdoutStderr, err := exec.CommandContext(
		ctx,
		nfqlbCmd,
		"deactivate",
		fmt.Sprintf("--index=%d", identifier-1),
		fmt.Sprintf("--shm=%s", n.name),
	).CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// SetFlow -
// SetFlow adds a Flow selecting the associated shared mem lb representing a Stream
//
// Note:
// It also weeds out /0 IP ranges and 'any' port filters if possible to simplify the
// config and improve match performance.
func (n *nfqlb) SetFlow(flow *nspAPI.Flow) error {
	ctx := context.TODO()
	args := []string{
		"flow-set",
		fmt.Sprintf("--name=%v-%v", n.name, flow.GetName()),
		fmt.Sprintf("--target=%v", n.name),
		fmt.Sprintf("--prio=%v", flow.GetPriority()),
		fmt.Sprintf("--protocols=%v", strings.Join(flow.GetProtocols(), ",")),
	}
	if vips := flow.GetVips(); vips != nil {
		dsts := []string{}
		for _, vip := range vips {
			dsts = append(dsts, vip.GetAddress())
		}
		args = append(args, fmt.Sprintf("--dsts=%v", strings.Join(dsts, ",")))
	}
	if srcs := flow.GetSourceSubnets(); srcs != nil && !n.anyIPRange(srcs) {
		args = append(args, fmt.Sprintf("--srcs=%v", strings.Join(srcs, ",")))
	}
	if dports := flow.GetDestinationPortRanges(); dports != nil && !n.anyPortRange(dports) {
		args = append(args, fmt.Sprintf("--dports=%v", strings.Join(dports, ",")))
	}
	if sports := flow.GetSourcePortRanges(); sports != nil && !n.anyPortRange(sports) {
		args = append(args, fmt.Sprintf("--sports=%v", strings.Join(sports, ",")))
	}
	if byteMatches := flow.GetByteMatches(); byteMatches != nil {
		args = append(args, fmt.Sprintf("--match=%v", strings.Join(byteMatches, ",")))
	}

	cmd := exec.CommandContext(
		ctx,
		nfqlbCmd,
		args...,
	)

	logrus.Debugf("cmd: %v", cmd.String())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// DeleteFlow -
// DeleteFlow removes a Flow which served to select the associated shared mem lb representing a Stream
func (n *nfqlb) DeleteFlow(flow *nspAPI.Flow) error {
	ctx := context.TODO()
	args := []string{
		"flow-delete",
		fmt.Sprintf("--name=%v-%v", n.name, flow.GetName()),
	}

	cmd := exec.CommandContext(
		ctx,
		nfqlbCmd,
		args...,
	)

	logrus.Debugf("cmd: %v", cmd.String())
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%v; %s", err, stdoutStderr)
	}
	return err
}

// anyPortRange -
// Returns true if ANY of the possible input port ranges cover all the possible ports (0-65535)
func (n *nfqlb) anyPortRange(ports []string) bool {
	for _, port := range ports {
		if port == MAXPORTRANGE {
			return true
		}
	}
	return false
}

// anyIPRange -
// Returns true if ALL the IP ranges are /0
//
// Note:
// IPv4 and IPv6 ranges can be mixed in both Meridio Flows and nfqlb Flows.
// When specified, nfqlb Flow's srcs/dsts selectors will NOT match IP version
// for whom no IP range is set.
func (n *nfqlb) anyIPRange(ips []string) bool {
	for _, ip := range ips {
		s := strings.Split(ip, "/")
		if len(s) == 1 { // should never not happen, nfqlb expects subnet mask...
			return false
		}
		mask, err := strconv.Atoi(s[1])
		if err != nil {
			// resort to stating input IP ranges are not 'any' (worst case the flow rule won't get simplified)
			return false
		}
		if mask != 0 {
			// non zero subnet mask i.e. not 'any' range
			return false
		}
	}

	return true
}
