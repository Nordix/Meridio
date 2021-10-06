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

package frontend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/frontend/internal/bird"
	"github.com/nordix/meridio/cmd/frontend/internal/connectivity"
	"github.com/nordix/meridio/cmd/frontend/internal/env"
	"github.com/nordix/meridio/cmd/frontend/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
)

// FrontEndService -
func NewFrontEndService(c *env.Config) *FrontEndService {
	logrus.Infof("NewFrontEndService")

	conn, err := grpc.Dial(c.NSPService, grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		logrus.Errorf("grpc.Dial err: %v", err)
	}
	targetRegistryClient := nspAPI.NewTargetRegistryClient(conn)

	frontEndService := &FrontEndService{
		vips:     []string{},
		gateways: []*utils.Gateway{},
		gatewayNamesByFamily: map[int]map[string]string{
			syscall.AF_INET:  {},
			syscall.AF_INET6: {},
		},
		vrrps:                c.VRRPs,
		birdConfPath:         c.BirdConfigPath,
		birdConfFile:         c.BirdConfigPath + "/bird-fe-meridio.conf",
		kernelTableId:        c.TableID,
		extInterface:         c.ExternalInterface,
		localASN:             c.LocalAS,
		remoteASN:            c.RemoteAS,
		localPortBGP:         c.BGPLocalPort,
		remotePortBGP:        c.BGPRemotePort,
		holdTimeBGP:          c.BGPHoldTime,
		ecmp:                 c.ECMP,
		dropIfNoPeer:         c.DropIfNoPeer,
		logBird:              c.LogBird,
		reconfCh:             make(chan struct{}),
		targetRegistryClient: targetRegistryClient,
		advertiseVIP:         false,
		logNextMonitorStatus: true,
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
	gateways             []*utils.Gateway
	gatewayNamesByFamily map[int]map[string]string
	gwMu                 sync.Mutex
	cfgMu                sync.Mutex
	monitorMu            sync.Mutex
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
	dropIfNoPeer         bool
	logBird              bool
	reconfCh             chan struct{}
	advertiseVIP         bool
	logNextMonitorStatus bool
	targetRegistryClient nspAPI.TargetRegistryClient
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

// WaitStart -
// Wait until BIRD started by checking birdc availability
func (fes *FrontEndService) WaitStart(ctx context.Context) error {
	lp, err := bird.LookupCli()
	if err != nil {
		return fmt.Errorf("WaitStart: birdc not found! (%v)", err)
	}
	var timeoutScale time.Duration = 1000000 // 1 ms
	i := 1
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("WaitStart: Shutting down")
			return nil
		case <-time.After(timeoutScale * time.Nanosecond): //timeout
		}

		err := bird.CheckCli(ctx, lp)
		if err != nil {
			if i <= 10 {
				timeoutScale += 10000000 // 10 ms
			}
			logrus.Debugf("WaitStart: not ready yet...\n%v", err)
		} else {
			break
		}
	}
	return nil
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
	logrus.Debugf("SetNewConfig")

	fes.cfgMu.Lock()
	defer fes.cfgMu.Unlock()

	switch c := c.(type) {
	case []*nspAPI.Attractor:
		// FE watches 1 Attractor that it is associated with
		if len(c) == 1 {
			logrus.Debugf("SetNewConfig: Attractor: %v", c)
			c := c[0]
			if err := fes.setVIPs(c.Vips, &configChange); err != nil {
				return err
			}
			if err := fes.setGateways(c.Gateways, &configChange); err != nil {
				return err
			}
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
	lp, err := bird.LookupCli()
	if err != nil {
		logrus.Errorf("ReloadConfig: Birdc not found!")
		return err
	}

	logForced := func() bool {
		fes.monitorMu.Lock()
		defer fes.monitorMu.Unlock()
		forced := fes.logNextMonitorStatus
		fes.logNextMonitorStatus = false
		return forced
	}

	//linkCh := make(chan string, 1)
	go func() {
		var once sync.Once
		//defer close(linkCh)
		// status of external connectivity; requires 1 GW per IP family if configured to be reachable
		init, noConnectivity := true, true
		connectivityMap := map[string]bool{}
		delay := 3 * time.Second // when started grant more time to write the whole config (minimize intial link flapping)
		_ = fes.WaitStart(ctx)
		for {
			select {
			case <-ctx.Done():
				logrus.Infof("Monitor: shutting down")
				return
			case <-time.After(delay): //timeout
				delay = 1 * time.Second
			}

			forced := logForced() // force status logs after config updates
			protocolOut, err := bird.ShowProtocolSessions(ctx, lp, `NBR-*`)
			if err != nil {
				logrus.Warnf("Monitor: %v: %v", err, protocolOut)
				//Note: if birdc is not yet running, no need to bail out
				//linkCh <- "Failed to fetch protocol status"
			} else if strings.Contains(protocolOut, bird.NoProtocolsLog) {
				if !noConnectivity || init {
					_ = denounceFrontend(fes.targetRegistryClient)
					noConnectivity = true
					connectivityMap = map[string]bool{}
					logrus.Warnf("Monitor: %v", protocolOut)
					//linkCh <- "No protocols match"
				}
			} else {
				bfdOut, err := bird.ShowBfdSessions(ctx, lp, `NBR-BFD`)
				if err != nil {
					logrus.Warnf("Monitor: %v: %v", err, bfdOut)
					//Note: if birdc is not yet running, no need to bail out
					//linkCh <- "Failed to fetch bfd status"
					break
				}
				// determine status of gateways and connectivity
				status := fes.parseStatusOutput(protocolOut, bfdOut)

				// Gateway availibility notifications
				// XXX: in case denounceFrontend/announceFrontend would block the thread for too long
				// (no NSP is listening etc.), and it is a problem, move them to dedicated go thread
				// TODO: maybe move logic to separate function
				if status.NoConnectivity() {
					// although configured at least one IP family has no connectivity
					// Note: deanounce FE even if just started (init); container might have crashed
					if !noConnectivity || init {
						noConnectivity = true
						if err := denounceFrontend(fes.targetRegistryClient); err != nil {
							logrus.Infof("FrontEndService: failed to denounce frontend connectivity (err: %v)", err)
						}
						fes.denounceVIP()
					}
				} else {
					if noConnectivity {
						noConnectivity = false
						if err := announceFrontend(fes.targetRegistryClient); err != nil {
							logrus.Infof("FrontEndService: failed to announce frontend connectivity (err: %v)", err)
						}
						fes.announceVIP()
					}
				}

				// Logging
				// log gateway connectivity information upon config or protocol status changes
				if forced || !reflect.DeepEqual(connectivityMap, status.StatusMap()) {
					if status.AnyGatewayDown() {
						logrus.Warnf("Monitor: (status=%v) %v", status.Status(), status.Log())
					} else {
						logrus.Infof("Monitor: (status=%v) %v", status.Status(), status.Log())
					}

				}
				connectivityMap = status.StatusMap()

				// TODO: ugly
				once.Do(func() {
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
// Note: In case of Static the related BFD session's state is also verified,
// as that is not refelected by the Static protocol's state.
func (fes *FrontEndService) parseStatusOutput(output string, bfdOutput string) *connectivity.ConnectivityStatus {
	cs := connectivity.NewConnectivityStatus()
	fes.gwMu.Lock()
	defer fes.gwMu.Unlock()

	if len(fes.gatewayNamesByFamily[syscall.AF_INET]) == 0 {
		cs.SetNoConfig(syscall.AF_INET)
	}
	if len(fes.gatewayNamesByFamily[syscall.AF_INET6]) == 0 {
		cs.SetNoConfig(syscall.AF_INET6)
	}

	bird.ParseProtocols(output, cs.Logp(), func(name string, options ...bird.Option) {
		//logrus.Infof("parseStatusOutput: name: %v", name)
		ip, family, ok := fes.getGatewayIPByName(name)
		if !ok { // no configured gateway found for the name
			return
		}

		// extend protocol options with external inteface, gateway ip, available bfd sessions
		opts := append([]bird.Option{
			bird.WithInterface(fes.extInterface),
			bird.WithNeighbor(ip),
			bird.WithBfdSessions(bfdOutput),
		}, options...)
		// check if protocol session is down
		if bird.ProtocolDown(bird.NewProtocol(opts...)) {
			cs.SetGatewayDown(name) // neighbor protocol down
		} else {
			cs.SetGatewayUp(name, family) // neighbor protocol up
		}
	})

	return cs
}

// VerifyConfig -
// Verify BIRD config file
//
// prerequisite: BIRD must be running so that birdc could talk to it
func (fes *FrontEndService) VerifyConfig(ctx context.Context) error {
	lp, err := bird.LookupCli()
	if err != nil {
		logrus.Errorf("ReloadConfig: Birdc not found!")
		return err
	} else {
		stringOut, err := bird.Verify(ctx, lp, fes.birdConfFile)
		if err != nil {
			logrus.Errorf("VerifyConfig: %v: %v", err, stringOut)
			return err
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

	// Have to add BFD protocol so that BGP or STATIC could ask for a BFD session
	*conf += "protocol bfd 'NBR-BFD' {\n"
	*conf += "\tinterface \"" + fes.extInterface + "\";\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter matching default IPv4, IPv6 routes
	*conf += "filter default_rt {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	/* // Filter matching default IPv4 routes
	*conf += "filter default_ipv4 {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter matching default IPv6 routes
	*conf += "filter default_ipv6 {\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n" */

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
	*conf += "\tbfd off;\n" // can be enabled per protocol session i.e. per gateway
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
	fes.gatewayNamesByFamily[syscall.AF_INET] = map[string]string{}
	fes.gatewayNamesByFamily[syscall.AF_INET6] = map[string]string{}

	for _, gw := range fes.gateways {
		if utils.IsIPv6(gw.Address) || utils.IsIPv4(gw.Address) {
			if gw.Protocol != "bgp" && gw.Protocol != "static" {
				logrus.Infof("writeConfigGW: Unkown gateway protocol %v", gw.Protocol)
				continue
			}

			nbr := strings.Split(gw.Address, "/")[0]
			name := `NBR-` + gw.Name
			family := "ipv4"
			afFamily := syscall.AF_INET
			allNet := "0.0.0.0/0"
			if utils.IsIPv6(gw.Address) {
				family = "ipv6"
				afFamily = syscall.AF_INET6
				allNet = "0::/0"
			}
			// save neighbor IP to be used by parse functions
			fes.gatewayNamesByFamily[afFamily][name] = nbr

			// TODO: const
			switch gw.Protocol {
			case "bgp":
				{
					ipv := "\t" + family + " {\n"
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_rt;\n"
					}
					ipv += "\t\texport filter cluster_e_static;\n"
					ipv += "\t};\n"

					*conf += "protocol bgp '" + name + "' from LINK {\n"
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
			case "static":
				{
					bfd := ""
					if gw.BFD {
						bfd = " bfd"
					}
					// default route via the gateway through the external interface
					ro := "\troute " + allNet + " via " + nbr + "%'" + fes.extInterface + "'" + bfd + ";\n"
					ipv := "\t" + family + " {\n"
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_rt;\n"
					}
					ipv += "\t};\n"

					*conf += "protocol static '" + name + "' {\n"
					*conf += ipv
					*conf += ro
					*conf += "}\n"
					*conf += "\n"
				}
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
		eFilter = "filter default_rt"
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
		eFilter = "filter default_rt"
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
		if utils.IsIPv6(vip) {
			// IPv6
			//v6 += "\troute " + vip + " blackhole;\n"
			v6 += "\troute " + vip + " via \"lo\";\n"
		} else if utils.IsIPv4(vip) {
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
		if utils.IsIPv6(ip) || utils.IsIPv4(ip) {
			*conf += "protocol static {\n"
			if utils.IsIPv4(ip) {
				*conf += "\tipv4;\n"
				*conf += "\troute 0.0.0.0/0 via " + strings.Split(ip, "/")[0] + "%'" + fes.extInterface + "' onlink;\n"
			} else if utils.IsIPv6(ip) {
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

	if err := bird.Run(ctx, fes.birdConfFile, fes.logBird); err != nil {
		logrus.Errorf("FrontEndService: BIRD err: \"%v\"", err)
		errCh <- err
	}
}

// reconfigurationAgent -
// Reconfigure BIRD when ordered to
//
// prerequisite: BIRD must be started
func (fes *FrontEndService) reconfigurationAgent(ctx context.Context, reconfCh <-chan struct{}, errCh chan<- error) {
	lp, err := bird.LookupCli()
	if err != nil {
		err := fmt.Errorf("reconfigurationAgent: birdc not found! (%v)", err)
		logrus.Errorf("%v", err)
		errCh <- err
	} else {
		if err := fes.WaitStart(ctx); err != nil {
			errCh <- err
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
					} else {
						fes.monitorMu.Lock()
						fes.logNextMonitorStatus = true
						fes.monitorMu.Unlock()
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
	stringOut, err := bird.Configure(ctx, path, fes.birdConfFile)
	if err != nil {
		logrus.Errorf("reconfigure: %v: %v", err, stringOut)
		return err
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
	case []*nspAPI.Vip:
		list := []string{}
		for _, vip := range vips {
			list = append(list, vip.GetAddress())
		}
		added, removed = utils.Difference(fes.vips, list)
		logrus.Debugf("SetVIPs: got: %+v, (added: %v, removed: %v)", vips, added, removed)
		fes.vips = list
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
	case []*nspAPI.Gateway:
		list := utils.ConvertGateways(gateways)
		logrus.Debugf("SetGateways: \ngot: %+v \nhave: %+v", list, fes.gateways)
		if utils.DiffGateways(list, fes.gateways) {
			logrus.Debugf("SetGateways: config changed")
			fes.gateways = list
			*change = true
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
			rule.Src = utils.StrToIPNet(vip)

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
			rule.Src = utils.StrToIPNet(vip)

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

func (fes *FrontEndService) getGatewayIPByName(name string) (string, int, bool) {
	ok := false
	addr := ""
	family := syscall.AF_UNSPEC

	if addr, ok = fes.gatewayNamesByFamily[syscall.AF_INET][name]; ok {
		family = syscall.AF_INET
	} else if addr, ok = fes.gatewayNamesByFamily[syscall.AF_INET6][name]; ok {
		family = syscall.AF_INET6
	}

	return addr, family, ok
}

/* func (fes *FrontEndService) checkGatewayByName(name string, ipfamily int) (string, bool) {
	switch ipfamily {
	case syscall.AF_INET:
		fallthrough
	case syscall.AF_INET6:
		addr, ok := fes.gatewayNamesByFamily[ipfamily][name]
		return addr, ok
	default:
		logrus.Infof("checkGatewayByName: unsupported ipfamily: %v (name: %v)", ipfamily, name)
		return "", false
	}
} */

//-------------------------------------------------------------------------------------------

// TODO: what to do once Static+BFD gets introduced? Probably do nothing...
// TODO: when there's only static, no need to play with announce/denounceVIP...

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
