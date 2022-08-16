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

package health

const (
	DefaultURL string = "unix:///tmp/health.sock"
)

const (
	Startup   string = "Startup"
	Readiness string = "Readiness"
	Liveness  string = "Liveness"
)

const (
	IPAMSvc              string = "IPAM"
	IPAMCliSvc           string = "IPAMCli"
	NSPCliSvc            string = "NSPCli"
	EgressSvc            string = "Egress"
	NSMEndpointSvc       string = "NSMEndpoint"
	TargetRegistryCliSvc string = "TargetRegistryCli"
	StreamSvc            string = "Stream"
	FlowSvc              string = "Flow"
)

var LBReadinessServices []string = []string{NSPCliSvc, NSMEndpointSvc, EgressSvc, StreamSvc, FlowSvc}
var FEReadinessServices []string = []string{NSPCliSvc, TargetRegistryCliSvc, EgressSvc}
var ProxyReadinessServices []string = []string{IPAMCliSvc, NSPCliSvc, NSMEndpointSvc, EgressSvc}
var IPAMReadinessServices []string = []string{NSPCliSvc, IPAMSvc}
