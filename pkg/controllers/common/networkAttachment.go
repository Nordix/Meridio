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

package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const (
	NetworkAttachmentAnnot = "k8s.v1.cni.cncf.io/networks"
)

// NetworkAttachmentSelector -
// Represents a selector for a json formattad Network Attachment Annotation.
// (Not using meridiov1alpha1 NetworkAttachmentSpec to keep compatibility even
// if the meridio API gets extended/changed in the future.)
//
// If required, can be replaced with e.g. 3rd party NetworkSelectionElement:
// https://github.com/k8snetworkplumbingwg/network-attachment-definition-client/blob/master/pkg/apis/k8s.cni.cncf.io/v1/types.go#L135
type NetworkAttachmentSelector struct {
	Name             string `json:"name,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
	InterfaceRequest string `json:"interface,omitempty"`
}

// refer to meridiov1alpha1 NetworkAttachmentSpec for clues about the patterns
var networkAnnotRegexItems = map[string]*regexp.Regexp{
	"name":      regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`),
	"namespace": regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}?[a-z0-9])?$`),
	"interface": regexp.MustCompile(`^[^\s:/]{1,13}$`),
}

func parsePodNetworkObjectName(podnetwork string) (string, string, string, error) {
	var netNsName string
	var netIfName string
	var networkName string

	slashItems := strings.Split(podnetwork, "/")
	if len(slashItems) == 2 {
		netNsName = strings.TrimSpace(slashItems[0])
		networkName = slashItems[1]
	} else if len(slashItems) == 1 {
		networkName = slashItems[0]
	} else {
		return "", "", "", errors.Errorf("parsePodNetworkObjectName: Invalid network object (failed at '/')")
	}

	atItems := strings.Split(networkName, "@")
	networkName = strings.TrimSpace(atItems[0])
	if len(atItems) == 2 {
		netIfName = strings.TrimSpace(atItems[1])
	} else if len(atItems) != 1 {
		return "", "", "", errors.Errorf("parsePodNetworkObjectName: Invalid network object (failed at '@')")
	}

	// Check and see if each item matches the specification
	allItems := []struct {
		value string
		reg   *regexp.Regexp
	}{
		{networkName, networkAnnotRegexItems["name"]},
		{netNsName, networkAnnotRegexItems["namespace"]},
		{netIfName, networkAnnotRegexItems["interface"]},
	}

	for _, v := range allItems {
		matched := v.reg.MatchString(v.value)
		if !matched && len([]rune(v.value)) > 0 {
			return "", "", "", errors.Errorf(fmt.Sprintf("parsePodNetworkObjectName: Failed to parse: one or more items did not match comma-delimited format. Mismatch @ '%v'", v.value))
		}
	}

	return netNsName, networkName, netIfName, nil
}

// GetNetworkAnnotation -
// Parses k8s.v1.cni.cncf.io/networks annotations, and fills in the namespace information
// if missing with defaultNamespace.
// Understands both json and <namespace>/<network name>@<ifname> format.
func GetNetworkAnnotation(networks, defaultNamespace string) ([]*NetworkAttachmentSelector, error) {
	var networksSelElems []*NetworkAttachmentSelector

	if networks == "" {
		return nil, nil
	}

	if strings.ContainsAny(networks, "[{\"") {
		if err := json.Unmarshal([]byte(networks), &networksSelElems); err != nil {
			return nil, errors.Errorf("parsePodNetworkAnnotation: failed to parse pod Network Attachment Selection Annotation JSON format: %v", err)
		}
	} else {
		// Comma-delimited list of network attachment object names
		for _, item := range strings.Split(networks, ",") {
			// Remove leading and trailing whitespace.
			item = strings.TrimSpace(item)

			// Parse network name (i.e. <namespace>/<network name>@<ifname>)
			netNsName, networkName, netIfName, err := parsePodNetworkObjectName(item)
			if err != nil {
				return nil, errors.Errorf("parsePodNetworkAnnotation: %v", err)
			}

			networksSelElems = append(networksSelElems, &NetworkAttachmentSelector{
				Name:             networkName,
				Namespace:        netNsName,
				InterfaceRequest: netIfName,
			})
		}
	}

	for _, n := range networksSelElems {
		if n.Namespace == "" {
			n.Namespace = defaultNamespace
		}
	}

	return networksSelElems, nil
}

// MakeNetworkAttachmentSpecMap -
// Creates a map from list of NetworkAttachmentSelectors.
func MakeNetworkAttachmentSpecMap(list []*NetworkAttachmentSelector) map[NetworkAttachmentSelector]*NetworkAttachmentSelector {
	m := make(map[NetworkAttachmentSelector]*NetworkAttachmentSelector)
	for _, item := range list {
		m[*item] = item
	}
	return m
}
