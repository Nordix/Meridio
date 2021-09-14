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
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nordix/meridio/pkg/configuration"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/nordix/meridio-operator/controllers/config"
)

const (
	IPv4Up       = uint64(1 << iota)           // FE has IPv4 external connectivity
	IPv6Up                                     // FE has IPv6 external connectivity
	NoIPv4Config                               // No IPv4 Gateways configured
	NoIPv6Config                               // No IPv6 Gateways configured
	AnyGWDown                                  // Not all configured gateways are available
	Up           = IPv4Up | IPv6Up             // FE has IPv4 and IPv6 external connectivity
	NoConfig     = NoIPv4Config | NoIPv6Config // No Gateways configured at all
)

type connectivityStatus struct {
	status uint64
	log    string
}

func (cs *connectivityStatus) noConnectivity() bool {
	return cs.status&NoConfig != 0 || (cs.status&NoIPv4Config == 0 && cs.status&IPv4Up == 0) || (cs.status&NoIPv6Config == 0 && cs.status&IPv6Up == 0)
}

func (cs *connectivityStatus) anyGatewayDown() bool {
	return cs.status&AnyGWDown != 0
}

// FrontEndService -
func NewFrontEndService(c *Config) *FrontEndService {
	logrus.Infof("NewFrontEndService")

	frontEndService := &FrontEndService{
		vips:     []string{},
		gateways: &config.GatewayConfig{},
		gatewayNamesByFamily: map[string]map[string]struct{}{
			"ipv4": {},
			"ipv6": {},
		},
		vrrps:         c.VRRPs,
		birdConfPath:  c.BirdConfigPath,
		birdConfFile:  c.BirdConfigPath + "/bird-fe-meridio.conf",
		kernelTableId: c.TableID,
		extInterface:  c.ExternalInterface,
		localASN:      c.LocalAS,
		remoteASN:     c.RemoteAS,
		localPortBGP:  c.BGPLocalPort,
		remotePortBGP: c.BGPRemotePort,
		holdTimeBGP:   c.BGPHoldTime,
		ecmp:          c.ECMP,
		bfd:           c.BFD,
		dropIfNoPeer:  c.DropIfNoPeer,
		logBird:       c.LogBird,
		reconfCh:      make(chan struct{}),
		nspService:    c.NSPService,
		advertiseVIP:  false,
	}

	if len(frontEndService.vrrps) > 0 {
		// When using static default routes there's no need for blackhole routes...
		frontEndService.dropIfNoPeer = false
	}

	return frontEndService
}

// FrontEndService -
type FrontEndService struct {
	vips                 []string
	gateways             *config.GatewayConfig
	gatewayNamesByFamily map[string]map[string]struct{}
	gwMu                 sync.Mutex
	cfgMu                sync.Mutex
	vrrps                []string
	birdConfPath         string
	birdConfFile         string
	kernelTableId        int
	extInterface         string
	localASN             string
	remoteASN            string
	localPortBGP         string
	remotePortBGP        string
	holdTimeBGP          string
	ecmp                 bool
	bfd                  bool
	dropIfNoPeer         bool
	logBird              bool
	reconfCh             chan struct{}
	nspService           string
	advertiseVIP         bool
}

// CleanUp -
// Basic clean-up of FrontEndService
func (fes *FrontEndService) CleanUp() {
	logrus.Infof("FrontEndService: CleanUp")

	close(fes.reconfCh)
	_ = fes.RemoveVIPRules()
}

func (fes *FrontEndService) Init() error {
	logrus.Infof("FrontEndService: Init")

	return fes.writeConfig()
}

// Start -
// Start BIRD with the generated config
func (fes *FrontEndService) Start(ctx context.Context) <-chan error {
	logrus.Infof("FrontEndService: Starting")

	errCh := make(chan error, 1)
	go fes.start(ctx, errCh)
	go fes.reconfigurationAgent(ctx, fes.reconfCh, errCh)

	return errCh
}

// AddVipRules -
// Add src based routing rules for VIP addresses (pointing to the routing table BIRD shall sync to)
func (fes *FrontEndService) AddVIPRules() error {
	return fes.setVIPRules(fes.vips, []string{})
}

// RemoveVipRules -
// Remove all the previously installed VIP src routing rules
func (fes *FrontEndService) RemoveVIPRules() error {
	return fes.setVIPRules([]string{}, fes.vips)
}

// SetNewConfig -
// Adjust BIRD config on the fly
func (fes *FrontEndService) SetNewConfig(c interface{}) error {
	configChange := false
	logrus.Infof("SetNewConfig")

	fes.cfgMu.Lock()
	defer fes.cfgMu.Unlock()

	switch c := c.(type) {
	case *configuration.OperatorConfig:
		if err := fes.setVIPs(c.VIPs, &configChange); err != nil {
			return err
		}
		// TODO: it could make sense to increase the bird monitor interval,
		// or to start some timer etc. that while not expired would make
		// the BIRD monitoring results to be ignored -> it would allow on
		// the fly added gateways to connect without the risk of disturbing
		// the "general availability" of the particular FE (the other option
		// is to be strict, and upon a new gateway for a new ipfamily denounce
		// FE availablity)
		// XXX: should we even bother? The NSM mesh between proxies and lbs
		// is either ipv4,ipv6 or dualstack.
		if err := fes.setGateways(c.GWs, &configChange); err != nil {
			return err
		}
	default:
		logrus.Infof("SetNewConfig: config format not known")
	}

	if configChange {
		return fes.promoteConfigNoLock()
	}

	return nil
}

// Monitor -
// Check bgp prorocols' status by periodically querying birdc.
// - Log changes in availablity/connectivity.
// - Reflect external connectivity through NSP to be used by LBs
// to steer outbound traffic.
// - Withdraw VIPs in case FE is considered down, in order to not
// attract traffic through possible available links if any. More like
// VIPs must be added to BIRD only if the frontend is considered up.
// (Note: IPv4/IPv6 backplane not separated).
func (fes *FrontEndService) Monitor(ctx context.Context) error {
	lp, err := exec.LookPath("birdc")
	if err != nil {
		logrus.Errorf("ReloadConfig: Birdc not found!")
		return err
	}

	//linkCh := make(chan string, 1)
	go func() {
		var once sync.Once
		//defer close(linkCh)
		extConnsOK, init := true, true
		noConnectivity := true // status of external connectivity; requires 1 GW per IP family if configured to be reachable
		for {
			select {
			case <-ctx.Done():
				logrus.Infof("Monitor: shutting down")
				return
			case <-time.After(5 * time.Second): //timeout
			}

			arg := `"NBR-*"`
			cmd := exec.CommandContext(ctx, lp, "show", "protocols", "all", arg)
			//cmd := exec.CommandContext(ctx, lp, "show", "protocols", arg)
			stdoutStderr, err := cmd.CombinedOutput()
			stringOut := string(stdoutStderr)
			if err != nil {
				logrus.Warnf("Monitor: %v: %v", err, stringOut)
				//Note: if birdc is not yet running, no need to bail out
				//linkCh <- "Failed to fetch protocol status"
			} else if strings.Contains(stringOut, "No protocols match") {
				if extConnsOK {
					_ = denounceFrontend(fes.nspService)
					extConnsOK = false
					logrus.Warnf("Monitor: %v", stringOut)
					//linkCh <- "No protocols match"
				}
			} else {
				status := fes.parseStatusOutput(stringOut)
				//logrus.Debugf("Monitor: status %v", status.status)

				// Gateway availibility notifications

				// XXX: in case denounceFrontend/announceFrontend would block the thread for too long
				// (no NSP is listening etc.), and it is a problem, move them to dedicated go thread
				// TODO: maybe move logic to separate function
				if status.noConnectivity() {
					// although configured at least one IP family has no connectivity
					// Note: deanounce FE even if just started (init); container might have
					// crashed
					if !noConnectivity || init {
						noConnectivity = true
						if err := denounceFrontend(fes.nspService); err != nil {
							logrus.Infof("FrontEndService: failed to denounce frontend connectivity (err: %v)", err)
						}
						fes.denounceVIP()
					}
				} else {
					if noConnectivity {
						noConnectivity = false
						if err := announceFrontend(fes.nspService); err != nil {
							logrus.Infof("FrontEndService: failed to announce frontend connectivity (err: %v)", err)
						}
						fes.announceVIP()
					}
				}

				// Logging

				if extConnsOK && status.anyGatewayDown() {
					extConnsOK = false
					if !init {
						logrus.Warnf("Monitor: (status=%v) %v", status.status, status.log)
					}
				} else if status.anyGatewayDown() {
					extConnsOK = false
					if !init {
						logrus.Debugf("Monitor: (status=%v) %v", status.status, status.log)
					}
				} else if !extConnsOK && !status.anyGatewayDown() {
					extConnsOK = true
					if !init {
						logrus.Infof("Monitor: (status=%v) %v", status.status, status.log)
					}
				}
				// TODO: ugly
				once.Do(func() {
					logrus.Infof("Monitor: (status=%v) %v", status.status, status.log)
					init = false
				})
			}
		}
	}()
	return nil
}

// parseStatusOutput -
// Parse birdc status output to determine external connectivity.
// Consider only BIRD proto sessions belonging to valid gayeways.
// Note: External connectivity is not yet separated for IPv4 and IPv6.
// The frontend is considered down, if it has no established gateways at
// all, or there are no established gateways for an IP family although
// it has gateways configured.
func (fes *FrontEndService) parseStatusOutput(output string) *connectivityStatus {
	cs := &connectivityStatus{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	reBGP := regexp.MustCompile(`NBR-`)
	reBIRD := regexp.MustCompile(`BIRD|Name\s+Proto`)

	fes.gwMu.Lock()
	defer fes.gwMu.Unlock()

	if len(fes.gatewayNamesByFamily["ipv4"]) == 0 {
		cs.status |= NoIPv4Config
	}
	if len(fes.gatewayNamesByFamily["ipv6"]) == 0 {
		cs.status |= NoIPv6Config
	}

	for scanner.Scan() {
		if ok := reBGP.MatchString(scanner.Text()); ok {
			cs.log += scanner.Text()
			// get name of the session, and check if belongs to a configured gateway
			if fields := strings.Fields(scanner.Text()); len(fields) > 0 {
				sgw := fields[0]

				if !strings.Contains(scanner.Text(), "Established") {
					if ok := fes.checkGatewayByName(sgw, ""); ok {
						// configured session not Established; mark it (used by logging)
						cs.status |= AnyGWDown
					}
				} else {
					if cs.status&Up != Up {
						if fes.checkGatewayByName(sgw, "ipv4") {
							// at least 1 configured ipv4 gw up
							cs.status |= IPv4Up
						} else if fes.checkGatewayByName(sgw, "ipv6") {
							// at least 1 configured ipv6 gw up
							cs.status |= IPv6Up
						}
					}
				}
			}
		} else if strings.Contains(scanner.Text(), `Neighbor address`) {
			cs.log += scanner.Text() + "\n"
		} else if ok := reBIRD.MatchString(scanner.Text()); ok {
			cs.log += scanner.Text() + "\n"
		}
	}

	return cs
}

// VerifyConfig -
// Verify BIRD config file
//
// prerequisite: BIRD must be running so that birdc could talk to it
func (fes *FrontEndService) VerifyConfig(ctx context.Context) error {
	lp, err := exec.LookPath("birdc")
	if err != nil {
		logrus.Errorf("ReloadConfig: Birdc not found!")
		return err
	} else {
		arg := `"` + fes.birdConfFile + `"`
		cmd := exec.CommandContext(ctx, lp, "configure", "check", arg)
		stdoutStderr, err := cmd.CombinedOutput()
		stringOut := string(stdoutStderr)
		if err != nil {
			logrus.Errorf("VerifyConfig: %v: %v", err, stringOut)
			return err
		} else if !strings.Contains(stringOut, "Configuration OK") {
			logrus.Errorf("VerifyConfig: %v", stringOut)
			return errors.New("Verification failed")
		} else {
			logrus.Debugf("VerifyConfig: %v", stringOut)
			return nil
		}
	}
}

//-------------------------------------------------------------------------------------------

// promoteConfig -
// Write BIRD config and initiate BIRD reconfiguration
func (fes *FrontEndService) promoteConfig() error {
	fes.cfgMu.Lock()
	defer fes.cfgMu.Unlock()

	return fes.promoteConfigNoLock()
}

// promoteConfigNoLock -
func (fes *FrontEndService) promoteConfigNoLock() error {
	if err := fes.writeConfig(); err != nil {
		logrus.Fatalf("promoteConfig: Failed to generate config: %v", err)
		return err
	}
	// send signal to reconfiguration agent to apply the new config
	logrus.Debugf("promoteConfig: Singnal config change")
	fes.reconfCh <- struct{}{}
	return nil
}

// writeConfig -
// Create BIRD config file
//
// Can be used both for the initial config and for later changes as well. (BIRD can
// reconfigure itself based on loading the new config file - refer to reconfigurationAgent())
func (fes *FrontEndService) writeConfig() error {
	file, err := os.Create(fes.birdConfFile)
	if err != nil {
		logrus.Errorf("FrontEndService: failed to create %v, err: %v", fes.birdConfFile, err)
		return err
	}
	defer file.Close()

	//conf := "include \"bird-common.conf\";\n"
	//conf += "\n"
	conf := ""
	fes.writeConfigBase(&conf)
	hasVIP4, hasVIP6 := fes.writeConfigVips(&conf)
	if len(fes.vrrps) > 0 {
		fes.writeConfigVRRPs(&conf, hasVIP4, hasVIP6)
	} else if fes.dropIfNoPeer {
		fes.writeConfigDropIfNoPeer(&conf, hasVIP4, hasVIP6)
	}
	fes.writeConfigKernel(&conf, hasVIP4, hasVIP6)
	fes.writeConfigGW(&conf)

	logrus.Infof("FrontEndService: BIRD config generated")
	logrus.Debugf("\n%v", conf)
	_, err = file.WriteString(conf)
	if err != nil {
		logrus.Errorf("FrontEndService: failed to write %v, err: %v", fes.birdConfFile, err)
	}

	return err
}

// writeConfigBase -
// Common part of BIRD config
func (fes *FrontEndService) writeConfigBase(conf *string) {
	*conf += "log syslog all;\n"
	*conf += "log \"/var/log/bird.log\" { debug, trace, info, remote, warning, error, auth, fatal, bug };\n"
	if fes.logBird {
		*conf += "log stderr all;\n"
	} else {
		*conf += "log stderr { error, fatal, bug, warning };\n"
	}
	*conf += "\n"

	// The Device protocol is not a real routing protocol. It does not generate any
	// routes and it only serves as a module for getting information about network
	// interfaces from the kernel. It is necessary in almost any configuration.
	*conf += "protocol device {\n"
	*conf += "}\n"
	*conf += "\n"

	// Have to add BFD protocol so that BGP could ask for a BFD session
	*conf += "protocol bfd {\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter matching default IPv4 routes
	*conf += "filter default_v4 {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter matching default IPv6 routes
	*conf += "filter default_v6 {\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// filter telling what BGP nFE can send to BGP GW peers
	// hint: only the VIP addresses
	//
	// Note: Since VIPs are configured as static routes in BIRD, there's
	// no point maintaining complex v4/v6 filters. Such filters would require
	// updates upon changes related to VIP addresses anyways...
	*conf += "filter cluster_e_static {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then reject;\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then reject;\n"
	*conf += "\tif source = RTS_STATIC && dest != RTD_BLACKHOLE then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// BGP protocol template
	*conf += "template bgp LINK {\n"
	*conf += "\tdebug {events, states, interfaces};\n"
	*conf += "\tdirect;\n"
	*conf += "\thold time " + fes.holdTimeBGP + ";\n"
	bfdSwitch := "off"
	if fes.bfd {
		bfdSwitch = "on"
	}
	*conf += "\tbfd " + bfdSwitch + ";\n"
	*conf += "\tgraceful restart off;\n"
	*conf += "\tsetkey off;\n"
	*conf += "\tipv4 {\n"
	*conf += "\t\timport none;\n"
	*conf += "\t\texport none;\n"
	// advertise this router as next-hop
	*conf += "\t\tnext hop self;\n"
	*conf += "\t};\n"

	*conf += "\tipv6 {\n"
	*conf += "\t\timport none;\n"
	*conf += "\t\texport none;\n"
	// advertise this router as next-hop
	*conf += "\t\tnext hop self;\n"
	*conf += "\t};\n"
	*conf += "}\n"
	*conf += "\n"
}

// writeConfigGW -
// Create BGP proto part of the BIRD config for each gateway to connect with them
//
// BGP is restricted to the external interface.
// Only VIP related routes are announced to peer, and only default routes are accepted.
//
// Note: When VRRP IPs are configured, BGP sessions won't import any routes from external
// peers, as external routes are going to be taken care of by static default routes (VRRP IPs
// as next hops).
func (fes *FrontEndService) writeConfigGW(conf *string) {
	fes.gwMu.Lock()
	defer fes.gwMu.Unlock()
	fes.gatewayNamesByFamily["ipv4"] = map[string]struct{}{}
	fes.gatewayNamesByFamily["ipv6"] = map[string]struct{}{}

	for _, gw := range fes.gateways.Gateways {
		if isIPv6(gw.Address) || isIPv4(gw.Address) {
			// TODO: const
			if gw.Protocol == "bgp" {
				ipv := ""
				if isIPv4(gw.Address) {
					fes.gatewayNamesByFamily["ipv4"]["NBR-"+gw.Name] = struct{}{}

					ipv += "\tipv4 {\n"
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_v4;\n"
					}
					ipv += "\t\texport filter cluster_e_static;\n"
					ipv += "\t};\n"
				} else if isIPv6(gw.Address) {
					fes.gatewayNamesByFamily["ipv6"]["NBR-"+gw.Name] = struct{}{}

					ipv = "\tipv6 {\n"
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_v6;\n"
					}
					ipv += "\t\texport filter cluster_e_static;\n"
					ipv += "\t};\n"
				}
				nbr := strings.Split(gw.Address, "/")[0]
				*conf += "protocol bgp 'NBR-" + gw.Name + "' from LINK {\n"
				*conf += "\tinterface \"" + fes.extInterface + "\";\n"

				// session specific BGP params
				localASN := fes.localASN
				localPort := fes.localPortBGP
				remoteASN := fes.remoteASN
				remotePort := fes.remotePortBGP
				if gw.LocalASN != 0 {
					localASN = strconv.FormatUint(uint64(gw.LocalASN), 10)
				}
				if gw.LocalPort != 0 {
					localPort = strconv.FormatUint(uint64(gw.LocalPort), 10)
				}
				if gw.RemoteASN != 0 {
					remoteASN = strconv.FormatUint(uint64(gw.RemoteASN), 10)
				}
				if gw.RemotePort != 0 {
					remotePort = strconv.FormatUint(uint64(gw.RemotePort), 10)
				}
				*conf += "\tlocal port " + localPort + " as " + localASN + ";\n"
				*conf += "\tneighbor " + nbr + " port " + remotePort + " as " + remoteASN + ";\n"

				if gw.BFD {
					*conf += "\tbfd on;\n"
				}

				if gw.HoldTime != 0 {
					*conf += "\thold time " + strconv.FormatUint(uint64(gw.HoldTime), 10) + ";\n"
				}
				*conf += ipv
				*conf += "}\n"
				*conf += "\n"
			}
		}
	}
}

// writeConfigKernel -
// Create kernel proto part of the BIRD config
//
// Kernel proto is used to sync default routes learnt from BGP peer into
// local network stack (to the specified routing table).
// Note: No need to sync learnt default routes to stack, in case there are
// no VIPs configured for the particular IP family.
func (fes *FrontEndService) writeConfigKernel(conf *string, hasVIP4 bool, hasVIP6 bool) {
	eFilter := "none"
	if hasVIP4 {
		eFilter = "filter default_v4"
	}

	*conf += "protocol kernel {\n"
	*conf += "\tipv4 {\n"
	*conf += "\t\timport none;\n"
	*conf += "\t\texport " + eFilter + ";\n"
	*conf += "\t};\n"
	*conf += "\tkernel table " + strconv.FormatInt(int64(fes.kernelTableId), 10) + ";\n"
	if fes.ecmp {
		*conf += "\tmerge paths on;\n"
	}
	if fes.dropIfNoPeer {
		// Setting the metric for the default blackhole route must be supported,
		// which requires the kernel proto's metric to be set to zero.
		//
		// "Metric 0 has a special meaning of undefined metric, in which either OS default is used,
		// or per-route metric can be set using krt_metric attribute. Default: 32. "
		*conf += "\tmetric 0;\n"
	}
	*conf += "}\n"
	*conf += "\n"

	if hasVIP6 {
		eFilter = "filter default_v6"
	} else {
		eFilter = "none"
	}
	*conf += "protocol kernel {\n"
	*conf += "\tipv6 {\n"
	*conf += "\t\timport none;\n"
	*conf += "\t\texport " + eFilter + ";\n"
	*conf += "\t};\n"
	*conf += "\tkernel table " + strconv.FormatInt(int64(fes.kernelTableId), 10) + ";\n"
	if fes.ecmp {
		*conf += "\tmerge paths on;\n"
	}
	if fes.dropIfNoPeer {
		// Setting the metric for the default blackhole route must be supported,
		// which requires the kernel proto's metric to be set to zero.
		//
		// "Metric 0 has a special meaning of undefined metric, in which either OS default is used,
		// or per-route metric can be set using krt_metric attribute. Default: 32. "
		*conf += "\tmetric 0;\n"
	}
	*conf += "}\n"
	*conf += "\n"
}

// writeConfigVips -
// Create static routes for VIP addresses in BIRD config
//
// VIP addresses are configured as static routes in BIRD. They are
// only advertised to BGP peers and not synced into local network stack.
//
// Note: VIPs shall be advertised only if external connectivity is OK
func (fes *FrontEndService) writeConfigVips(conf *string) (hasVIP4, hasVIP6 bool) {
	v4, v6 := "", ""
	hasVIP4, hasVIP6 = false, false

	for _, vip := range fes.vips {
		if isIPv6(vip) {
			// IPv6
			//v6 += "\troute " + vip + " blackhole;\n"
			v6 += "\troute " + vip + " via \"lo\";\n"
		} else if isIPv4(vip) {
			// IPv4
			//v4 += "\troute " + vip + " blackhole;\n"
			v4 += "\troute " + vip + " via \"lo\";\n"
		}
	}

	if v4 != "" {
		hasVIP4 = true
		if fes.advertiseVIP {
			*conf += "protocol static VIP4 {\n"
			*conf += "\tipv4 { preference 110; };\n"
			*conf += v4
			*conf += "}\n"
			*conf += "\n"
		}
	}

	if v6 != "" {
		hasVIP6 = true
		if fes.advertiseVIP {
			*conf += "protocol static VIP6 {\n"
			*conf += "\tipv6 { preference 110; };\n"
			*conf += v6
			*conf += "}\n"
			*conf += "\n"
		}
	}
	return
}

// writeConfigDropIfNoPeer -
// Create static default blackhole routes, that will be synced to the routing table
// used by VIP src routing rules.
//
// The aim is to drop packets from VIP src addresses when no external gateways are
// available, when configured accordingly. (So that the POD's default route installed
// for the prrimary network couldn't interfere.)
// XXX: Default route of the primary network could still interfere e.g. when a VIP is
// removed; there can be a transient period of time when FE still gets outbound packets
// with the very VIP as src addr.
//
// Notes:
// - These routes are configured with the highest (linux) metric (-> lowest prio)
// - BIRD 2.0.7 has a strange behaviour that differs between IPv4 and IPv6:
//		- IPv4: - seemingly all the default routes are installed to the OS kernel routing
//                table including e.g. default blackhole routes with lower preference
//              - BIRD fails to remove all the BGP related routes when there's a BIRD
//                managed blackhole route for the same destination as well
//		- IPv6: default route with the highest preference gets installed to OS kernel routing table
func (fes *FrontEndService) writeConfigDropIfNoPeer(conf *string, hasVIP4 bool, hasVIP6 bool) {
	if hasVIP4 {
		*conf += "protocol static BH4 {\n"
		*conf += "\tipv4 { preference 0; };\n"
		*conf += "\troute 0.0.0.0/0 blackhole {\n"
		*conf += "\t\tkrt_metric=4294967295;\n"
		*conf += "\t};\n"
		*conf += "}\n"
		*conf += "\n"
	}

	if hasVIP6 {
		*conf += "protocol static BH6 {\n"
		*conf += "\tipv6 { preference 0; };\n"
		*conf += "\troute 0::/0 blackhole {\n"
		*conf += "\t\tkrt_metric=4294967295;\n"
		*conf += "\t};\n"
		*conf += "}\n"
		*conf += "\n"
	}
}

// writeConfigVRRPs -
// BIRD managed default static routes substituting other routing protocol related
// external routes.
func (fes *FrontEndService) writeConfigVRRPs(conf *string, hasVIP4, hasVIP6 bool) {
	for _, ip := range fes.vrrps {
		if isIPv6(ip) || isIPv4(ip) {
			*conf += "protocol static {\n"
			if isIPv4(ip) {
				*conf += "\tipv4;\n"
				*conf += "\troute 0.0.0.0/0 via " + strings.Split(ip, "/")[0] + "%'" + fes.extInterface + "' onlink;\n"
			} else if isIPv6(ip) {
				*conf += "\tipv6;\n"
				*conf += "\troute 0::/0 via " + strings.Split(ip, "/")[0] + "%'" + fes.extInterface + "' onlink;\n"
			}
			*conf += "}\n"
			*conf += "\n"
		}
	}
}

//-------------------------------------------------------------------------------------------

// start -
// Actually start BIRD process.
// Based on logBird settings stderr of the started BIRD process can be monitored,
// so that important log snippets get appended to the container's log.
func (fes *FrontEndService) start(ctx context.Context, errCh chan<- error) {
	logrus.Infof("FrontEndService: start (monitor BIRD logs=%v)", fes.logBird)
	defer close(errCh)
	defer logrus.Warnf("FrontEndService: Run fnished")

	if !fes.logBird {
		if stdoutStderr, err := exec.CommandContext(ctx, "bird", "-d", "-c", fes.birdConfFile).CombinedOutput(); err != nil {
			logrus.Errorf("FrontEndService: err: \"%v\", out: %s", err, stdoutStderr)
			errCh <- err
		}
	} else {
		cmd := exec.CommandContext(ctx, "bird", "-d", "-c", fes.birdConfFile)
		// get stderr pipe reader that will be connected with the process' stderr by Start()
		pipe, err := cmd.StderrPipe()
		if err != nil {
			logrus.Errorf("FrontEndService: stderr pipe err: \"%v\"", err)
			errCh <- err
			return
		}

		// Note: Probably not needed at all, as due to the use of CommandContext()
		// Start() would kill the process as soon context become done. Which should
		// lead to an EOF on stderr anyways.
		go func() {
			// make sure bufio Scan() can be breaked out from when context is done
			w, ok := cmd.Stderr.(*os.File)
			if !ok {
				// not considered a deal-breaker at the moment; see above note
				logrus.Debugf("FrontEndService: cmd.Stderr not *os.File")
				return
			}
			// when context is done, close File thus signalling EOF to bufio Scan()
			defer w.Close()
			<-ctx.Done()
			logrus.Infof("FrontEndService: context closed, terminate log monitoring...")
		}()

		// start the process (BIRD)
		if err := cmd.Start(); err != nil {
			logrus.Errorf("FrontEndService: start err: \"%v\"", err)
			errCh <- err
			return
		}

		// scan stderr of previously started process
		// Note: there could be other log-worthy printouts...
		scanner := bufio.NewScanner(pipe)
		reW := regexp.MustCompile(`Error|<ERROR>|<BUG>|<FATAL>|<WARNING>`)
		reI := regexp.MustCompile(`<INFO>|BGP session|Connected|Received:|Started|Neighbor|Startup delayed`)
		for scanner.Scan() {
			if ok := reW.MatchString(scanner.Text()); ok {
				logrus.Warnf("[bird] %v", scanner.Text())
			} else if ok := reI.MatchString(scanner.Text()); ok {
				logrus.Infof("[bird] %v", scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			logrus.Errorf("FrontEndService: scanner err: \"%v\"", err)
			errCh <- err
		}

		// wait until process concludes
		// (should only get here after stderr got closed or scanner returned error)
		if err := cmd.Wait(); err != nil {
			logrus.Errorf("FrontEndService: err: \"%v\"", err)
			errCh <- err
		}
	}
}

// reconfigurationAgent -
// Reconfigure BIRD when ordered to
//
// prerequisite: BIRD must be started
func (fes *FrontEndService) reconfigurationAgent(ctx context.Context, reconfCh <-chan struct{}, errCh chan<- error) {
	lp, err := exec.LookPath("birdc")
	if err != nil {
		logrus.Fatalf("reconfigurationAgent: birdc not found! (%v)", err)
	} else {
		var timeoutScale time.Duration = 1000000 // 1 ms
		i := 1
		// wait until BIRD has started
		for {
			select {
			case <-ctx.Done():
				logrus.Infof("reconfigurationAgent: Shutting down")
				return
			case <-time.After(timeoutScale * time.Nanosecond): //timeout
			}
			cmd := exec.CommandContext(ctx, lp, "show", "status")
			stdoutStderr, err := cmd.CombinedOutput()
			stringOut := string(stdoutStderr)
			if err != nil {
				if i <= 10 {
					timeoutScale += 10000000 // 10 ms
				}
				logrus.Debugf("reconfigurationAgent: not ready yet...\n%v: %v: %v", cmd.String(), err, stringOut)
			} else {
				break
			}
		}

		// listen for reconf signals
		logrus.Infof("reconfigurationAgent: Ready")
		for {
			select {
			case <-ctx.Done():
				logrus.Infof("reconfigurationAgent: Shutting down")
				return
			case _, ok := <-fes.reconfCh:
				if ok {
					if err := fes.reconfigure(ctx, lp); err != nil {
						logrus.Errorf("reconfigurationAgent: Failed to reconfigure BIRD (err: \"%v\")", err)
						errCh <- err
					}
				}
				// if not ok; clean-up was called closing reconfCh
				// if so, ctx.Done() should kick in as well...
			}
		}
	}
}

// reconfigure -
// Order reconfiguration of BIRD through birdc (i.e. apply new config file)
func (fes *FrontEndService) reconfigure(ctx context.Context, path string) error {
	arg := `"` + fes.birdConfFile + `"`
	cmd := exec.CommandContext(ctx, path, "configure", arg)
	stdoutStderr, err := cmd.CombinedOutput()
	stringOut := string(stdoutStderr)
	if err != nil {
		logrus.Errorf("reconfigure: %v: %v", err, stringOut)
		return err
	} else if !strings.Contains(stringOut, "Reconfiguration in progress") && !strings.Contains(stringOut, "Reconfigured") {
		logrus.Errorf("reconfigure: %v", stringOut)
		return errors.New("Reconfiguration failed")
	} else {
		logrus.Debugf("reconfigure: %v", stringOut)
		logrus.Infof("reconfigure: BIRD config applied")
		return nil
	}
}

//-------------------------------------------------------------------------------------------

// setVIPs -
// Adjust config to changes affecting VIP addresses
func (fes *FrontEndService) setVIPs(vips interface{}, change *bool) error {
	var added, removed []string
	switch vips := vips.(type) {
	case *config.VipConfig:
		if vips != nil {
			list := configuration.AddrListFromVipConfig(vips)
			added, removed = difference(fes.vips, list)
			logrus.Debugf("SetVIPs: got: %v, (added: %v, removed: %v)", vips.Vips, added, removed)
			fes.vips = list
		}
	default:
		logrus.Debugf("setVIPs: vips format not supported")
	}

	if len(added) > 0 || len(removed) > 0 {
		*change = true
		if err := fes.setVIPRules(added, removed); err != nil {
			logrus.Fatalf("SetVIPs: Failed to adjust VIP src routes: %v", err)
			return err
		}
	}

	return nil
}

// setGateways -
// Adjust config to changes affecting external gateway addresses
func (fes *FrontEndService) setGateways(gateways interface{}, change *bool) error {
	switch gateways := gateways.(type) {
	case *config.GatewayConfig:
		if gateways != nil {
			logrus.Debugf("SetGateways: \ngot: %v \nhave: %v", gateways.Gateways, fes.gateways.Gateways)
			if configuration.DiffGatewayConfig(gateways, fes.gateways) {
				fes.gateways.Gateways = gateways.Gateways
				*change = true
			}
		}
	default:
		logrus.Debugf("SetGateways: gateways format not supported")
	}

	return nil
}

// setVIPRules -
// Add/remove VIP src routing rules based on changes
func (fes *FrontEndService) setVIPRules(vipsAdded, vipsRemoved []string) error {
	if len(vipsAdded) > 0 || len(vipsRemoved) > 0 {
		handler, err := netlink.NewHandle()
		if err != nil {
			logrus.Errorf("setVIPRules: Failed to open handler: %v", err)
			return err
		}
		defer handler.Delete()

		for _, vip := range vipsRemoved {
			rule := netlink.NewRule()
			rule.Priority = 100
			rule.Table = fes.kernelTableId
			rule.Src = strToIPNet(vip)

			logrus.Infof("setVIPRules: [del]: %v", rule)
			if err := handler.RuleDel(rule); err != nil {
				logrus.Warnf("setVIPRules: Failed to remove rule: %v", err)
				//TODO: return with error unless error refers to ENOENT/ESRCH
			}
		}

		for _, vip := range vipsAdded {
			rule := netlink.NewRule()
			rule.Priority = 100
			rule.Table = fes.kernelTableId
			rule.Src = strToIPNet(vip)

			logrus.Infof("setVIPRules: [add]: %v", rule)
			if err := handler.RuleAdd(rule); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					logrus.Errorf("setVIPRules: Failed to add rule: %v", err)
					return err
				}
			}
		}
	}

	return nil
}

func (fes *FrontEndService) checkGatewayByName(name, ipfamily string) bool {
	switch ipfamily {
	case "ipv4":
		_, ok := fes.gatewayNamesByFamily["ipv4"][name]
		return ok
	case "ipv6":
		_, ok := fes.gatewayNamesByFamily["ipv6"][name]
		return ok
	case "":
		if _, ok := fes.gatewayNamesByFamily["ipv4"][name]; ok {
			return true
		} else if _, ok := fes.gatewayNamesByFamily["ipv6"][name]; ok {
			return true
		}
	default:
		logrus.Infof("checkGatewayByName: unsupported ipfamily: %v (name: %v)", ipfamily, name)
		return false
	}

	return false
}

//-------------------------------------------------------------------------------------------

// TODO: what to do once Static+BFD gets introduced? Probably do nothing...

func (fes *FrontEndService) announceVIP() {
	logrus.Infof("FrontEndService: Announce VIPs")
	fes.cfgMu.Lock()
	fes.advertiseVIP = true
	fes.cfgMu.Unlock()

	go func() {
		if err := fes.promoteConfig(); err != nil {
			logrus.Warnf("FrontEndService: err AnnounceVIPs: %v", err)
		}
	}()
}

func (fes *FrontEndService) denounceVIP() {
	logrus.Infof("FrontEndService: Denounce VIPs")

	if !func() bool {
		fes.cfgMu.Lock()
		defer fes.cfgMu.Unlock()
		advertised := fes.advertiseVIP
		if advertised {
			fes.advertiseVIP = false
		}
		return advertised
	}() {
		// no need to rewrite config, VIPs not advertised in old config
		logrus.Debugf("Denounce VIPs: VIPs were not advertised")
		return
	}

	go func() {
		if err := fes.promoteConfig(); err != nil {
			logrus.Warnf("FrontEndService: err DenounceVIPs: %v", err)
		}
	}()
}
