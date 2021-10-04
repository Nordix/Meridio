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

package main

import (
	"os"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/nsp"
	"github.com/sirupsen/logrus"
)

// Notify Meridio whether the particular frontend has external connectivity or not.
// Currently only LB-FE composite is supported. Therefore LBs are only interested in events
// related to the collocated FE (residing in the same POD).
//
// Hostname information is used to determine collocation, thus there's no need to announce IPs
// as of now. However NSP requires "IP information" for item comparision, but hostnames do fine
// as well (NSP won't check if it's a real IP).
//
// Note: Currently IPv4/IPv6 connectivity is not separated, as IPv4/IPv6 handling is not properly
// separated in Meridio when managing the NSM backplane either.

// TODO: add context to nspClient calls so that they could be cancelled
// TODO: must denounce FE upon its shutdown (make sure it won't block forever e.g. in case NSP is no longer available)
// TODO: NSP must be improved to somehow learn if source of a Register event has disappeared (check NSM registry for clue)
// (maybe introduce timed Register that requires registration)
// TODO: maybe introduce update target through new context keyword, indicating nsp to replace found item with new one
func announceFrontend(service string) error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(service)
	if err != nil {
		return err
	}
	hn, _ := os.Hostname()
	targetContext := map[string]string{
		nsp.Identifier.String(): hn,
	}
	logrus.Infof("announceFrontend: hostname: %v, targetType: %v, nsp-service: %v", hn, nspAPI.Target_FRONTEND, service)
	err = nspClient.RegisterWithType(nspAPI.Target_FRONTEND, []string{hn}, targetContext)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}

func denounceFrontend(service string) error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(service)
	if err != nil {
		return err
	}
	hn, _ := os.Hostname()
	targetContext := map[string]string{
		nsp.Identifier.String(): hn,
	}
	logrus.Infof("denounceFrontend: hostname: %v, targetType: %v, nsp-service: %v", hn, nspAPI.Target_FRONTEND, service)
	err = nspClient.UnregisterWithContext(nspAPI.Target_FRONTEND, []string{hn}, targetContext)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}
