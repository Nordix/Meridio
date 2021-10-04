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

package nfqlb

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
)

const (
	ownfw    = 0
	qlength  = 1024
	nfqlbCmd = "nfqlb"
)

type nfqlb struct {
	name    string
	m       int
	n       int
	nfqueue int
}

func New(name string, m int, n int, nfqueue int) (*nfqlb, error) {
	lb := &nfqlb{
		name:    name,
		m:       m,
		n:       n,
		nfqueue: nfqueue,
	}
	err := lb.configure()
	if err != nil {
		return nil, err
	}
	err = lb.desactivateAll()
	if err != nil {
		return nil, err
	}
	return lb, nil
}

func (n *nfqlb) Activate(identifier int) error {
	var stderr bytes.Buffer
	cmd := exec.Command(
		nfqlbCmd,
		"activate",
		fmt.Sprintf("--index=%d", identifier-1),
		fmt.Sprintf("--shm=%s", n.getTargetSHM()),
		strconv.Itoa(identifier),
	)
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%w; %v", err, stderr.String())
	}
	return nil
}

func (n *nfqlb) Deactivate(identifier int) error {
	var stderr bytes.Buffer
	cmd := exec.Command(
		nfqlbCmd,
		"deactivate",
		fmt.Sprintf("--index=%d", identifier-1),
		fmt.Sprintf("--shm=%s", n.getTargetSHM()),
	)
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%w; %v", err, stderr.String())
	}
	return nil
}

// https://github.com/Nordix/nfqueue-loadbalancer/blob/master/src/nfqlb/cmdLb.c#L162
func (n *nfqlb) Start() error {
	cmd := exec.Command(
		nfqlbCmd,
		"lb",
		// "--mtu=",
		// "--tun=",
		// "--reassembler=",
		fmt.Sprintf("--tshm=%s", n.getTargetSHM()),
		// --lbshm=",
		fmt.Sprintf("--queue=%d", n.nfqueue),
		fmt.Sprintf("--qlength=%d", qlength),
		// "--ft_shm=",
		// "--ft_size=",
		// "--ft_buckets=",
		// "--ft_frag=",
		// "--ft_ttl=",
	)
	return cmd.Start()
}

func (n *nfqlb) Stop() error {
	return nil
}

func (n *nfqlb) Delete() error {
	return nil
}

// https://github.com/Nordix/nfqueue-loadbalancer/blob/master/src/nfqlb/cmdShm.c#L31
func (n *nfqlb) configure() error {
	var stderr bytes.Buffer
	cmd := exec.Command(
		nfqlbCmd,
		"init",
		fmt.Sprintf("--ownfw=%d", ownfw),
		fmt.Sprintf("--shm=%s", n.getTargetSHM()),
		fmt.Sprintf("--M=%d", n.m),
		fmt.Sprintf("--N=%d", n.n),
	)
	cmd.Stderr = &stderr
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%w; %v", err, stderr.String())
	}
	err = n.desactivateAll()
	if err != nil {
		return err
	}
	return nil
}

func (n *nfqlb) desactivateAll() error {
	for i := 1; i <= n.n; i++ {
		err := n.Deactivate(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *nfqlb) getTargetSHM() string {
	return fmt.Sprintf("tshm-%s", n.name)
}
