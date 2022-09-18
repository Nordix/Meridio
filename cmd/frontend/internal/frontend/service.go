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

	"github.com/go-logr/logr"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/cmd/frontend/internal/bird"
	feConfig "github.com/nordix/meridio/cmd/frontend/internal/config"
	"github.com/nordix/meridio/cmd/frontend/internal/connectivity"
	"github.com/nordix/meridio/cmd/frontend/internal/secret"
	"github.com/nordix/meridio/cmd/frontend/internal/utils"
	"github.com/nordix/meridio/pkg/health"
	"github.com/nordix/meridio/pkg/k8s/watcher"
	"github.com/nordix/meridio/pkg/log"
	"github.com/nordix/meridio/pkg/retry"
	"github.com/vishvananda/netlink"
)

// FrontEndService -
func NewFrontEndService(ctx context.Context, c *feConfig.Config) *FrontEndService {
	logger := log.FromContextOrGlobal(ctx).WithValues("class", "FrontEndService")
	targetRegistryClient := nspAPI.NewTargetRegistryClient(c.NSPConn)

	birdConfFile := c.BirdConfigPath + "/bird-fe-meridio.conf"
	authCh := make(chan struct{}, 10)
	sdb := secret.NewDatabase(ctx, authCh)

	frontEndService := &FrontEndService{
		vips:     []string{},
		gateways: []*utils.Gateway{},
		gatewayNamesByFamily: map[int]map[string]*utils.Gateway{
			syscall.AF_INET:  {},
			syscall.AF_INET6: {},
		},
		vrrps:                c.VRRPs,
		birdConfPath:         c.BirdConfigPath,
		birdConfFile:         birdConfFile,
		birdCommSocket:       c.BirdCommunicationSock,
		birdLogFileSize:      c.BirdLogFileSize,
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
		nspEntryTimeout:      c.NSPEntryTimeout,
		routingService:       bird.NewRoutingService(ctx, c.BirdCommunicationSock, birdConfFile),
		namespace:            c.Namespace,
		secretDatabase:       sdb,
		secretManager:        watcher.NewObjectMonitorManager(ctx, c.Namespace, sdb, secret.CreateSecretInterface),
		authCh:               authCh,
		logger:               logger,
	}

	if len(frontEndService.vrrps) > 0 {
		// When using static default routes there's no need for blackhole routes...
		frontEndService.dropIfNoPeer = false
	}

	logger.Info("Created", "object", frontEndService)
	return frontEndService
}

// FrontEndService -
type FrontEndService struct {
	vips                 []string
	gateways             []*utils.Gateway
	gatewayNamesByFamily map[int]map[string]*utils.Gateway
	gwMu                 sync.Mutex
	cfgMu                sync.Mutex
	monitorMu            sync.Mutex
	vrrps                []string
	birdConfPath         string
	birdConfFile         string
	birdCommSocket       string
	birdLogFileSize      int
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
	nspEntryTimeout      time.Duration
	routingService       *bird.RoutingService
	namespace            string
	secretDatabase       secret.DatabaseInterface              // stores contents of Secrets referenced by configuration
	secretManager        watcher.ObjectMonitorManagerInterface // keeps track changes of Secrets
	authCh               chan struct{}                         // used by secretDatabase to signal updates to FE Service
	logger               logr.Logger
}

// CleanUp -
// Basic clean-up of FrontEndService
func (fes *FrontEndService) CleanUp() {
	fes.logger.Info("CleanUp")

	close(fes.reconfCh)
	close(fes.authCh)
	_ = fes.RemoveVIPRules()
}

func (fes *FrontEndService) Init() error {
	fes.logger.Info("Init")

	return fes.writeConfig()
}

// Start -
// Start BIRD with the generated config
func (fes *FrontEndService) Start(ctx context.Context, errCh chan<- error) {
	fes.logger.Info("Start")

	go fes.start(ctx, errCh)
	go fes.reconfigurationAgent(ctx, fes.reconfCh, errCh)
	go fes.authenticationAgent(ctx, errCh) // monitors updates for secrets of interest to trigger reconfiguration
}

// Stop -
// Stop BIRD (attempt graceful shutdown)
func (fes *FrontEndService) Stop(ctx context.Context) {
	fes.logger.Info("Stop")
	fes.stop(ctx)
}

// WaitStart -
// Wait until BIRD started by checking birdc availability
func (fes *FrontEndService) WaitStart(ctx context.Context) error {
	logger := fes.logger.WithValues("func", "WaitStart")
	lp, err := fes.routingService.LookupCli()
	if err != nil {
		return fmt.Errorf("routing service cli not found: %v", err)
	}
	var timeoutScale time.Duration = 1000000 // 1 ms
	i := 1
	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			return nil
		case <-time.After(timeoutScale * time.Nanosecond): //timeout
		}

		err := fes.routingService.CheckCli(ctx, lp)
		if err != nil {
			if i <= 10 {
				timeoutScale += 10000000 // 10 ms
			}
			logger.V(1).Info("not ready yet", "out", err)
		} else {
			break
		}
	}
	return nil
}

// RemoveVipRules -
// Remove all the previously installed VIP src routing rules
func (fes *FrontEndService) RemoveVIPRules() error {
	return fes.setVIPRules([]string{}, fes.vips)
}

// SetNewConfig -
// Adjust BIRD config on the fly
func (fes *FrontEndService) SetNewConfig(ctx context.Context, c interface{}) error {
	configChange := false
	fes.logger.V(1).Info("SetNewConfig")
	logger := fes.logger.WithValues("func", "SetNewConfig")

	fes.cfgMu.Lock()
	defer fes.cfgMu.Unlock()

	switch c := c.(type) {
	case []*nspAPI.Attractor:
		// FE watches 1 Attractor that it is associated with
		if len(c) == 1 {
			logger.V(1).Info("Attractor", "Attractor", c)
			c := c[0]
			if err := fes.setVIPs(c.Vips, &configChange); err != nil {
				return err
			}
			gwConfigChange := false
			if err := fes.setGateways(c.Gateways, &gwConfigChange); err != nil {
				return err
			}
			if gwConfigChange {
				logger.V(2).Info("Gateway configuration changed")
				fes.checkAuthentication(ctx)
			}
			configChange = configChange || gwConfigChange
		}
	default:
		logger.Info("Unknown format")
	}

	if configChange {
		return fes.promoteConfigNoLock(ctx)
	}

	return nil
}

// Monitor -
// Check bgp prorocols' status by periodically querying birdc.
// - Log changes in availablity/connectivity.
// - Reflect external connectivity through NSP to be used by LBs
// to steer outbound traffic.
// - Withdraw VIPs in case FE is considered down, in order to not
// attract traffic through other available links if any. More like
// VIPs must be added to BIRD only if the frontend is considered up.
// (Note: IPv4/IPv6 backplane not separated).
func (fes *FrontEndService) Monitor(ctx context.Context, errCh chan<- error) {
	logger := fes.logger.WithValues("func", "Monitor")
	lp, err := fes.routingService.LookupCli()
	if err != nil {
		errCh <- fmt.Errorf("routing service cli not found: %v", err)
		return
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
		var refreshCancel context.CancelFunc
		for {
			select {
			case <-ctx.Done():
				logger.Info("Shutting down")
				if refreshCancel != nil {
					refreshCancel()
				}
				return
			case <-time.After(delay): //timeout
				delay = 1 * time.Second
			}

			forced := logForced() // force status logs after config updates
			protocolOut, err := fes.routingService.ShowProtocolSessions(ctx, lp, `NBR-*`)
			if err != nil {
				logger.Info("protocol output", "err", err, "out", strings.Split(protocolOut, "\n"))
				//Note: if birdc is not yet running, no need to bail out
				//linkCh <- "Failed to fetch protocol status"
			} else if strings.Contains(protocolOut, bird.NoProtocolsLog) {
				if !noConnectivity || init {
					if refreshCancel != nil {
						refreshCancel()
					}
					_ = denounceFrontend(fes.targetRegistryClient)
					noConnectivity = true
					connectivityMap = map[string]bool{}
					logger.Info("protocol output", "out", protocolOut)
					//linkCh <- "No protocols match"
				}
			} else {
				bfdOut, err := fes.routingService.ShowBfdSessions(ctx, lp, `NBR-BFD`)
				if err != nil {
					logger.Info("BFD output", "err", err, "out", strings.Split(bfdOut, "\n"))
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
						health.SetServingStatus(ctx, health.EgressSvc, false)
						if refreshCancel != nil {
							refreshCancel()
						}
						if err := denounceFrontend(fes.targetRegistryClient); err != nil {
							logger.Error(err, "Denounce frontend connectivity")
						}
						fes.denounceVIP(ctx, errCh)
					}
				} else {
					if noConnectivity {
						noConnectivity = false
						health.SetServingStatus(ctx, health.EgressSvc, true)
						logger.Info("Announce frontend")
						if err := announceFrontend(fes.targetRegistryClient); err != nil {
							logger.Error(err, "Announce frontend connectivity")
						}
						// refresh NSP entry
						var refreshCtx context.Context
						refreshCtx, refreshCancel = context.WithCancel(ctx)
						go func() {
							_ = retry.Do(func() error {
								return announceFrontend(fes.targetRegistryClient)
							}, retry.WithContext(refreshCtx),
								retry.WithDelay(fes.nspEntryTimeout),
								retry.WithErrorIngnored())
						}()
						fes.announceVIP(ctx, errCh)
					}
				}

				// Logging
				// log gateway connectivity information upon config or protocol status changes
				if forced || !reflect.DeepEqual(connectivityMap, status.StatusMap()) {
					if status.AnyGatewayDown() {
						logger.Error(fmt.Errorf("gateway down"), "connectivity", "status", status.Status(), "out", strings.Split(status.Log(), "\n"))
					} else {
						logger.Info("connectivity", "status", status.Status(), "out", strings.Split(status.Log(), "\n"))
					}

				}
				connectivityMap = status.StatusMap()

				// TODO: ugly
				once.Do(func() {
					init = false
				})
			}
		}
		if refreshCancel != nil {
			refreshCancel()
		}
	}()
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
		gw, family := fes.getGatewayByName(name)
		if gw == nil { // no configured gateway found for the name
			return
		}

		// extend protocol options with external inteface, gateway ip, available bfd sessions,
		// and with info whether bfd is configured for the particular gateway
		opts := append([]bird.Option{
			bird.WithInterface(fes.extInterface),
			bird.WithNeighbor(gw.GetNeighbor()),
			bird.WithBfdSessions(bfdOutput),
			bird.WithBfd(gw.BFD),
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
	logger := fes.logger.WithValues("func", "VerifyConfig")
	lp, err := fes.routingService.LookupCli()
	if err != nil {
		return err
	} else {
		stringOut, err := fes.routingService.Verify(ctx, lp)
		if err != nil {
			return fmt.Errorf("%v; %v", err, stringOut)
		} else {
			logger.V(1).Info("OK", "out", strings.Split(stringOut, "\n"))
			return nil
		}
	}
}

//-------------------------------------------------------------------------------------------
// TODO: Try to detach writeConfig components specific to bird

// promoteConfig -
// Write BIRD config and initiate BIRD reconfiguration
func (fes *FrontEndService) promoteConfig(ctx context.Context) error {
	fes.cfgMu.Lock()
	defer fes.cfgMu.Unlock()

	return fes.promoteConfigNoLock(ctx)
}

// promoteConfigNoLock -
func (fes *FrontEndService) promoteConfigNoLock(ctx context.Context) error {
	if err := fes.writeConfig(); err != nil {
		return fmt.Errorf("error writing configuration: %v", err)
	}
	// send signal to reconfiguration agent to apply the new config
	fes.logger.V(1).Info("promote configuration change")
	select {
	case fes.reconfCh <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("context closed, abort promote configuration")
	}
	return nil
}

// writeConfig -
// Create BIRD config file
//
// Can be used both for the initial config and for later changes as well. (BIRD can
// reconfigure itself based on loading the new config file - refer to reconfigurationAgent())
func (fes *FrontEndService) writeConfig() error {
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

	routingConfig := bird.NewRoutingConfig(fes.birdConfFile)
	routingConfig.Append(conf)
	fes.logger.Info("routing configuration generated")
	fes.logger.V(1).Info("routing configuration", "config", strings.Split(routingConfig.String(), "\n"))

	return routingConfig.Apply()
}

// writeConfigBase -
// Common part of BIRD config
func (fes *FrontEndService) writeConfigBase(conf *string) {
	if fes.birdLogFileSize > 0 {
		// TODO: const or make them configurable
		logFile := "/var/log/bird.log"
		logFileBackup := "/var/log/bird.log.backup"
		*conf += fmt.Sprintf("log \"%s\" %v \"%s\" { debug, trace, info, remote, warning, error, auth, fatal, bug };\n",
			logFile, fes.birdLogFileSize, logFileBackup)
	}
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

	// Filter matching default IPv4, IPv6 routes
	*conf += "filter default_rt {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
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
	fes.gatewayNamesByFamily[syscall.AF_INET] = map[string]*utils.Gateway{}
	fes.gatewayNamesByFamily[syscall.AF_INET6] = map[string]*utils.Gateway{}
	bfdSpec := &utils.BfdSpec{}

	writeBfdSpec := func(conf *string, b *utils.BfdSpec) {
		prefix := "\t\t"
		if b != nil {
			if b.MinRx != 0 {
				*conf += fmt.Sprintf("%vmin rx interval %vms;\n", prefix, b.MinRx)
			}
			if b.MinTx != 0 {
				*conf += fmt.Sprintf("%vmin tx interval %vms;\n", prefix, b.MinTx)
			}
			if b.Multiplier != 0 {
				*conf += fmt.Sprintf("%vmultiplier %v;\n", prefix, b.Multiplier)
			}
		}
	}

	for _, gw := range fes.gateways {
		if af := gw.GetAF(); af == syscall.AF_INET || af == syscall.AF_INET6 {
			if gw.Protocol != "bgp" && gw.Protocol != "static" {
				fes.logger.Info("Unkown gateway protocol", "protocol", gw.Protocol)
				continue
			}

			nbr := gw.GetNeighbor()
			name := `NBR-` + gw.Name
			family := "ipv4"
			allNet := "0.0.0.0/0"
			if af == syscall.AF_INET6 {
				family = "ipv6"
				allNet = "0::/0"
			}
			// save gateway to be used by parse functions to lookup neighbor IP and BFD
			fes.gatewayNamesByFamily[af][name] = gw

			// TODO: const
			switch gw.Protocol {
			case "bgp":
				{
					ipv := fmt.Sprintf("\t%v {\n", family)
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_rt;\n"
					}
					ipv += "\t\texport filter cluster_e_static;\n"
					ipv += "\t};\n"

					// BGP authentication
					var password []byte = nil
					if gw.BgpAuth != nil && gw.BgpAuth.KeySource != "" {
						key, err := fes.secretDatabase.Load(fes.namespace, gw.BgpAuth.KeySource, gw.BgpAuth.KeyName)
						if err != nil {
							fes.logger.Info("Skip gateway", "name", gw.Name, "reason", err)
							continue
						}
						password = key
					}

					*conf += fmt.Sprintf("protocol bgp '%v' from LINK {\n", name)
					*conf += fmt.Sprintf("\tinterface \"%v\";\n", fes.extInterface)
					// session specific BGP params
					if password != nil {
						*conf += fmt.Sprintf("\tpassword \"%s\";\n", password)
					}
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
					*conf += fmt.Sprintf("\tlocal port %v as %v;\n", localPort, localASN)
					*conf += fmt.Sprintf("\tneighbor %v port %v as %v;\n", nbr, remotePort, remoteASN)
					if gw.BFD {
						*conf += "\tbfd {\n"
						writeBfdSpec(conf, gw.BfdSpec)
						*conf += "\t};\n"
					}
					if gw.HoldTime != 0 {
						*conf += fmt.Sprintf("\thold time %v;\n", gw.HoldTime)
					}
					*conf += ipv
					*conf += "}\n"
					*conf += "\n"
				}
			case "static":
				{
					bfd := ""
					if gw.BFD {
						if gw.BfdSpec != nil {
							bfdSpec = gw.BfdSpec
						}
						bfd = " bfd"
					}
					// default route via the gateway through the external interface
					ro := fmt.Sprintf("\troute %v via %v%%'%v'%v;\n", allNet, nbr, fes.extInterface, bfd)
					ipv := fmt.Sprintf("\t%v {\n", family)
					if len(fes.vrrps) > 0 {
						ipv += "\t\timport none;\n"
					} else {
						ipv += "\t\timport filter default_rt;\n"
					}
					ipv += "\t};\n"

					*conf += fmt.Sprintf("protocol static '%v' {\n", name)
					*conf += ipv
					*conf += ro
					*conf += "}\n"
					*conf += "\n"
				}
			}
		}
	}

	// Have to add BFD protocol so that BGP or STATIC could ask for a BFD session
	// Note: BIRD 2.0.8 does not support per peer BFD attributes for Static protocol,
	// thus only 1 BFD configuration per interface is possible
	*conf += "protocol bfd 'NBR-BFD' {\n"
	*conf += fmt.Sprintf("\tinterface \"%v\" {\n", fes.extInterface)
	writeBfdSpec(conf, bfdSpec)
	*conf += "\t};\n"
	*conf += "}\n"
}

// writeConfigKernel -
// Create kernel proto part of the BIRD config
//
// Kernel proto is used to sync default routes learnt from BGP peer into
// local network stack (to the specified routing table).
// Note: No need to sync learnt default routes to stack, in case there are
// no VIPs configured for the particular IP family.
func (fes *FrontEndService) writeConfigKernel(conf *string, hasVIP4, hasVIP6 bool) {
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
//   - IPv4: - seemingly all the default routes are installed to the OS kernel routing
//     table including e.g. default blackhole routes with lower preference
//   - BIRD fails to remove all the BGP related routes when there's a BIRD
//     managed blackhole route for the same destination as well
//   - IPv6: default route with the highest preference gets installed to OS kernel routing table
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
	fes.logger.Info("start routing service", "log", fes.logBird)
	defer fes.logger.Info("routing service stopped running")

	if err := fes.routingService.Run(ctx, fes.logBird); err != nil {
		select {
		case errCh <- fmt.Errorf("error running routing service; %v", err):
		case <-ctx.Done():
		}
	}
}

// stop -
// Actually stop BIRD process.
func (fes *FrontEndService) stop(ctx context.Context) {
	lp, err := fes.routingService.LookupCli()
	if err != nil {
		fes.logger.Info("routing service cli not found", "err", err)
		return
	}
	if err := fes.routingService.CheckCli(ctx, lp); err != nil {
		fes.logger.Info("routing service cli not running", "err", err)
		return
	}
	if err := fes.routingService.ShutDown(ctx, lp); err != nil {
		fes.logger.Info("failure during routing service shutdown", "err", err)
		return
	}
}

// reconfigurationAgent -
// Reconfigure BIRD when ordered to
//
// prerequisite: BIRD must be started
func (fes *FrontEndService) reconfigurationAgent(ctx context.Context, reconfCh <-chan struct{}, errCh chan<- error) {
	lp, err := fes.routingService.LookupCli()
	if err != nil {
		errCh <- fmt.Errorf("reconfiguration agent error: routing service cli not found: %v", err)
	} else {
		if err := fes.WaitStart(ctx); err != nil {
			errCh <- fmt.Errorf("reconfiguration agent error: %v", err)
			return
		}

		// listen for reconf signals
		fes.logger.Info("reconfiguration agent Ready")
		for {
			select {
			case <-ctx.Done():
				fes.logger.Info("reconfiguration agent shutting down")
				return
			case _, ok := <-fes.reconfCh:
				if ok {
					if err := fes.reconfigure(ctx, lp); err != nil {
						errCh <- fmt.Errorf("reconfiguration agent error: %v", err)
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
// Order reconfiguration of BIRD through birdc (i.e. apply new config)
func (fes *FrontEndService) reconfigure(ctx context.Context, path string) error {
	stringOut, err := fes.routingService.Configure(ctx, path)
	if err != nil {
		return err
	} else {
		fes.logger.V(1).Info("routing service reconfigured", "out", strings.Split(stringOut, "\n"))
		fes.logger.Info("routing service configuration applied")
		return nil
	}
}

// checkAuthentication -
// Checks BGP Gateways with authentication enabled, and informs secretManager
// which secret objects must be monitored to get hold of the authentication data
func (fes *FrontEndService) checkAuthentication(ctx context.Context) {
	sourceList := []string{}
	sources := map[string]struct{}{}
	for _, gw := range fes.gateways {
		if gw.Protocol != "bgp" || gw.BgpAuth == nil || gw.BgpAuth.KeySource == "" {
			continue
		}
		sources[gw.BgpAuth.KeySource] = struct{}{}
	}

	for source := range sources {
		sourceList = append(sourceList, source)
	}

	fes.secretManager.Manage(ctx, sourceList)
}

// authenticationAgent -
// Watches changes related to authentication storage sources to initiate
// reconfiguration. (Data might change, or the information might not be
// available yet at the time the Gateway configuration is received from NSP.)
func (fes *FrontEndService) authenticationAgent(ctx context.Context, errCh chan<- error) {
	checkf := func() {
		fes.logger.V(2).Info("authentication agent initiating reconfiguration")
		if err := fes.promoteConfig(ctx); err != nil {
			select {
			case errCh <- fmt.Errorf("authentication agent error: %v", err):
			case <-ctx.Done():
			}
		}
	}
	drainf := func(ctx context.Context) bool {
		// drain secretCh for 100 ms to be protected against bursts
		for {
			select {
			case <-fes.authCh:
			case <-time.After(100 * time.Millisecond):
				return true
			case <-ctx.Done():
				return false
			}
		}
	}

	for {
		select {
		case <-fes.authCh:
			if drainf(ctx) {
				checkf()
			}
		case <-ctx.Done():
			fes.logger.Info("authentication agent shutting down")
			return
		}
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
		fes.logger.V(1).Info("setVIPs", "got", vips, "added", added, "removed", removed)
		fes.vips = list
	default:
		fes.logger.Info("VIP configuration format not supported")
	}

	if len(added) > 0 || len(removed) > 0 {
		*change = true
		if err := fes.setVIPRules(added, removed); err != nil {
			log.Fatal(fes.logger, "Failed to adjust VIP src routes", "error", err) // TODO: no fatal burreed deep
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
		fes.logger.V(1).Info("setGateways", "got", list, "have", fes.gateways)
		if utils.DiffGateways(list, fes.gateways) {
			fes.gateways = list
			*change = true
		}
	default:
		fes.logger.V(1).Info("Gateway configuration format not supported")
	}

	return nil
}

// setVIPRules -
// Add/remove VIP src routing rules based on changes
func (fes *FrontEndService) setVIPRules(vipsAdded, vipsRemoved []string) error {
	logger := fes.logger.WithValues("func", "setVIPRules")
	if len(vipsAdded) > 0 || len(vipsRemoved) > 0 {
		handler, err := netlink.NewHandle()
		if err != nil {
			logger.Error(err, "open netlink handler")
			return err
		}
		defer handler.Close()

		for _, vip := range vipsRemoved {
			rule := netlink.NewRule()
			rule.Priority = 100
			rule.Table = fes.kernelTableId
			rule.Src = utils.StrToIPNet(vip)

			logger.Info("Remove VIP rule", "rule", rule)
			if err := handler.RuleDel(rule); err != nil {
				logger.Error(err, "Remove VIP rule")
				//TODO: return with error unless error refers to ENOENT/ESRCH
			}
		}

		for _, vip := range vipsAdded {
			rule := netlink.NewRule()
			rule.Priority = 100
			rule.Table = fes.kernelTableId
			rule.Src = utils.StrToIPNet(vip)

			logger.Info("Add VIP rule", "rule", rule)
			if err := handler.RuleAdd(rule); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					logger.Error(err, "Add VIP rule")
					return err
				}
			}
		}
	}

	return nil
}

func (fes *FrontEndService) getGatewayByName(name string) (*utils.Gateway, int) {
	family := syscall.AF_UNSPEC
	gw, ok := fes.gatewayNamesByFamily[syscall.AF_INET][name]
	if ok {
		family = syscall.AF_INET
	} else if gw, ok = fes.gatewayNamesByFamily[syscall.AF_INET6][name]; ok {
		family = syscall.AF_INET6
	}

	return gw, family
}

//-------------------------------------------------------------------------------------------

// TODO: what to do once Static+BFD gets introduced? Probably do nothing...
// TODO: when there's only static, no need to play with announce/denounceVIP...

func (fes *FrontEndService) announceVIP(ctx context.Context, errCh chan<- error) {
	fes.logger.Info("announceVIP")
	fes.cfgMu.Lock()
	fes.advertiseVIP = true
	fes.cfgMu.Unlock()

	go func() {
		if err := fes.promoteConfig(ctx); err != nil {
			errCh <- fmt.Errorf("announceVIP error: %v", err)
		}
	}()
}

func (fes *FrontEndService) denounceVIP(ctx context.Context, errCh chan<- error) {
	fes.logger.Info("denounceVIP")

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
		fes.logger.V(1).Info("denounceVIP: no VIPs advertised")
		return
	}

	go func() {
		if err := fes.promoteConfig(ctx); err != nil {
			errCh <- fmt.Errorf("denounceVIP error: %v", err)
		}
	}()
}
