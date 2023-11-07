/*
Copyright (c) 2021-2023 Nordix Foundation

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

const rulePriorityVIP int = 100

// FrontEndService -
func NewFrontEndService(ctx context.Context, c *feConfig.Config, gatewayMetrics *GatewayMetrics) *FrontEndService {
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
		config:               c,
		gatewayMetrics:       gatewayMetrics,
	}

	if len(frontEndService.vrrps) > 0 {
		// When using static default routes there's no need for blackhole routes...
		frontEndService.dropIfNoPeer = false
	}

	gatewayMetrics.RoutingService = frontEndService.routingService

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
	config               *feConfig.Config
	gatewayMetrics       *GatewayMetrics
}

func (fes *FrontEndService) GetRoutingService() *bird.RoutingService {
	return fes.routingService
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
			fes.gatewayMetrics.Set(c.Gateways)
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
// - Log statistics related to routes managed by the routing suite.
func (fes *FrontEndService) Monitor(ctx context.Context, errCh chan<- error) {
	logger := fes.logger.WithValues("func", "Monitor")

	var (
		// hasConnectivity indicates external connectivity
		// requires 1 GW per IP family if configured to be reachable
		hasConnectivity bool
		// sessionErrors may be temporary
		sessionErrors int
		// lastStatusMap is the last statusMap
		lastStatusMap map[string]bool
		// delay between checks
		delay time.Duration
		// refreshCancel cancel periodic announcement to NSP
		refreshCancel context.CancelFunc
		// denounce indicates that the FE shall be denounced in the NSP
		denounce bool = true // Always denounce on container start
		// cancelRoute is used to cancel an old route checking operation
		cancelRoute context.CancelFunc
		// routeStats holds routing suite route statistics
		routeStats *RouteStats = NewRouteStats()
	)

	go func() {
		// Wait until BIRD started
		if err := fes.WaitStart(ctx); err != nil {
			logger.Error(err, "WaitStart")
		}
		lp, err := fes.routingService.LookupCli()
		if err != nil {
			// shoudn't fail if WaitStart() was ok
			errCh <- fmt.Errorf("routing service cli not found: %v", err)
			return
		}

		for {
			if denounce || sessionErrors > fes.config.MaxSessionErrors {
				denounce = false
				sessionErrors = 0
				hasConnectivity = false
				lastStatusMap = nil
				delay = fes.config.DelayNoConnectivity

				if refreshCancel != nil {
					refreshCancel()
					refreshCancel = nil
				}
				ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*30)
				if err := denounceFrontend(ctxTimeout, fes.targetRegistryClient); err != nil {
					logger.Error(err, "denounceFrontend")
				}
				cancel()
				health.SetServingStatus(ctx, health.EgressSvc, false)
				fes.denounceVIP(ctx, errCh)
			}

			select {
			case <-ctx.Done():
				logger.Info("Shutting down")
				if refreshCancel != nil {
					refreshCancel()
				}
				if hasConnectivity {
					ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()
					if err := denounceFrontend(ctxTimeout, fes.targetRegistryClient); err != nil {
						logger.Error(err, "Last denounceFrontend")
					}
				}
				return
			case <-time.After(delay): //timeout
				delay = fes.config.DelayConnectivity
			}

			// These errors may be temporary, so don't denounce immediately
			protocolOut, err := fes.routingService.ShowProtocolSessions(ctx, lp, `NBR-*`)
			if err != nil {
				logger.Error(err, "protocol output", "out", strings.Split(protocolOut, "\n"))
				sessionErrors++
				continue
			}
			bfdOut, err := fes.routingService.ShowBfdSessions(ctx, lp, `NBR-BFD`)
			if err != nil {
				logger.Error(err, "BFD output", "out", strings.Split(bfdOut, "\n"))
				sessionErrors++
				continue
			}
			sessionErrors = 0

			if strings.Contains(protocolOut, bird.NoProtocolsLog) {
				logger.Info("protocol output", "out", protocolOut)
				denounce = true
				continue
			}

			// determine status of gateways and connectivity
			status := fes.parseStatusOutput(protocolOut, bfdOut)

			// Gateway availibility notifications
			if status.NoConnectivity() {
				// although configured at least one IP family has no connectivity
				if hasConnectivity {
					denounce = true
				}
			} else {
				if !hasConnectivity {
					hasConnectivity = true
					health.SetServingStatus(ctx, health.EgressSvc, true)
					logger.Info("Announce frontend")
					ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*30)
					if err := announceFrontend(ctxTimeout, fes.targetRegistryClient); err != nil {
						logger.Error(err, "Announce frontend connectivity")
					}
					cancel()
					// Refresh NSP entry even if announceFrontend() above fails.
					// The FE still has external connectivity and should keep trying to
					// inform NSP about it.
					var refreshCtx context.Context
					refreshCtx, refreshCancel = context.WithCancel(ctx)
					go func() {
						// refresh
						_ = retry.Do(func() error {
							// guarantee connection with server (if announce timed out FE could retry right away)
							_ = retry.Do(func() error {
								ctxTimeout, cancel := context.WithTimeout(refreshCtx, time.Second*30)
								defer cancel()
								return announceFrontend(ctxTimeout, fes.targetRegistryClient)
							}, retry.WithContext(refreshCtx),
								retry.WithDelay(time.Second))

							return nil
						}, retry.WithContext(refreshCtx),
							retry.WithDelay(fes.nspEntryTimeout),
							retry.WithErrorIngnored())
					}()
					fes.announceVIP(ctx, errCh)
				}
			}

			// Logging
			// log gateway connectivity information upon config or protocol status changes
			fes.monitorMu.Lock()
			logForced := fes.logNextMonitorStatus
			fes.logNextMonitorStatus = false
			fes.monitorMu.Unlock()
			if logForced || !reflect.DeepEqual(lastStatusMap, status.StatusMap()) {
				if status.AnyGatewayDown() {
					logger.Error(fmt.Errorf("gateway down"), "connectivity", "status", status.ToString(), "out", strings.Split(status.Log(), "\n"))
				} else {
					logger.Info("connectivity", "status", status.ToString(), "out", strings.Split(status.Log(), "\n"))
				}

			}
			lastStatusMap = status.StatusMap()

			// Check number of routes
			// Note: Do not block the connectivity monitoring loop! BIRD blocks
			// on the operation while the routes are processed.
			if cancelRoute != nil {
				cancelRoute()
			}
			go func() {
				ctx, cancel := context.WithCancel(logr.NewContext(ctx, logger))
				cancelRoute = cancel
				defer cancelRoute()

				fes.checkRoutes(ctx, routeStats, lp)
			}()
		} // for {
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

		// extend protocol options with external interface, gateway ip, available bfd sessions,
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
	fes.logger.V(1).Info("config", "config", strings.Split(routingConfig.String(), "\n"))

	return routingConfig.Apply()
}

// writeConfigBase -
// Common part of BIRD config
func (fes *FrontEndService) writeConfigBase(conf *string) {
	if fes.birdLogFileSize > 0 {
		*conf += fmt.Sprintf("log \"%s\" %v \"%s\" { %s };\n",
			bird.LogFilePath, fes.birdLogFileSize, bird.BackupLogFilePath, bird.LogClasses)
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

	// Filter matches default IPv4, IPv6 routes
	*conf += "filter default_rt {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter matches default IPv4, IPv6 routes and routes originating from BGP protocol.
	// Intended usage is to control which routes are accepted by BGP import (i.e.
	// from BGP peers), and which routes the kernel protocol is allowed to export
	// into OS routing table.
	// At the end these routes will be used for cluster breakout.
	*conf += "filter cluster_breakout {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then accept;\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then accept;\n"
	*conf += "\tif source = RTS_BGP then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// Filter tells what local BGP can send to external BGP peers
	// hint: only the VIP addresses
	//
	// Note: Since VIPs are configured as static routes in BIRD, there's
	// no point maintaining complex v4/v6 filters. Such filters would require
	// updates upon changes related to VIP addresses anyways...
	*conf += "filter cluster_access {\n"
	*conf += "\tif ( net ~ [ 0.0.0.0/0 ] ) then reject;\n"
	*conf += "\tif ( net ~ [ 0::/0 ] ) then reject;\n"
	*conf += "\tif source = RTS_STATIC && dest != RTD_BLACKHOLE then accept;\n"
	*conf += "\telse reject;\n"
	*conf += "}\n"
	*conf += "\n"

	// BGP protocol templates (IPv4 and IPv6)
	// Separate templates ensure that multi-protocol extension capability is
	// only advertised for a single protocol when a Gateway is created from
	// any of the templates.
	bgpTemplate := func(af int, name string) string {
		template := fmt.Sprintf("template bgp %s {\n", name)
		template += "\tdebug {events, states, interfaces};\n"
		template += "\tdirect;\n"
		template += "\thold time " + fes.holdTimeBGP + ";\n"
		template += "\tbfd off;\n" // can be enabled per protocol session i.e. per gateway
		template += "\tgraceful restart off;\n"
		template += "\tsetkey off;\n"
		if af == syscall.AF_INET {
			template += "\tipv4 {\n"
		} else {
			template += "\tipv6 {\n"
		}
		template += "\t\timport none;\n"
		template += "\t\texport none;\n"
		// advertise this router as next-hop
		template += "\t\tnext hop self;\n"
		template += "\t};\n"
		template += "}\n"
		return template
	}
	*conf += bgpTemplate(syscall.AF_INET, bird.BGPTemplateIPv4)
	*conf += "\n"
	*conf += bgpTemplate(syscall.AF_INET6, bird.BGPTemplateIPv6)
	*conf += "\n"
}

// writeConfigGW -
// Creates BGP proto for BIRD config for each gateway configured to use BGP protocol.
// Creates Static proto for BIRD config for each gateway configured to use Static.
//
// BGP is restricted to the external interface. Only VIP related routes are announced
// to BGP peer, and both default and non-default routes are accepted from peer.
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
						ipv += "\t\timport filter cluster_breakout;\n"
					}
					ipv += "\t\texport filter cluster_access;\n"
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

					// chose BGP template based on IP protocol version
					*conf += fmt.Sprintf("protocol bgp '%v' from %s {\n",
						name,
						func() string {
							if af == syscall.AF_INET {
								return bird.BGPTemplateIPv4
							}
							return bird.BGPTemplateIPv6
						}(),
					)
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
					// Represented by a Static BIRD proto instance with a default route pointing to the gateway.
					// The filter controls pushing the default route to the BIRD routing table (master4/6).
					// (default route via the gateway through the external interface)
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
// Kernel proto is used to push both routes learnt from external BGP peers,
// and default routes available via Static(+BFD) gateway configs into
// local network stack of OS (into the kernel routing table configured).
// (Note: No need to sync learnt "outbond" routes to stack, in case
// no VIPs are configured for the particular IP family.)
func (fes *FrontEndService) writeConfigKernel(conf *string, hasVIP4, hasVIP6 bool) {
	eFilter := "none"
	if hasVIP4 {
		eFilter = "filter cluster_breakout"
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
	*conf += "}\n"
	*conf += "\n"

	eFilter = "none"
	if hasVIP6 {
		eFilter = "filter cluster_breakout"
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
// Create static default blackhole routes, that will be pushed to an OS kernel
// routing table used by VIP src routing rules.
//
// A secondary pair of BIRD kernel protocols are in place to push drop routes to a
// separate OS kernel routing table. The aim is to prevent misrouting of outbound
// VIP traffic due to POD default routes, in case no cluster breakout route has matched.
// For example none of the gateways are up, or BGP peers announced routes only for certain
// subnets.
//
// XXX: Default route of the primary network could still interfere e.g. when a VIP is
// removed; there can be a transient period of time when FE still gets outbound packets
// with the very VIP as src addr. (Could be avoided e.g. by marking packets entering via
// NSM intefaces and prohibiting their forwarding to the primary network.)
//
// (The reason for not using a common routing table is that a common table lead
// to problems irrespective of the fact whether metric in kernel proto was set to zero
// or not. If set to non-zero, there's  a transient period between removing old default
// routes and inserting new drop route. While setting the metric to zero causes BGP
// route withdrawal problems for IPv4, while in case of IPv6 the blackhole route is
// not added to OS kernel if there's another default routes with lower metric available.)
func (fes *FrontEndService) writeConfigDropIfNoPeer(conf *string, hasVIP4 bool, hasVIP6 bool) {
	if hasVIP4 {
		*conf += "ipv4 table drop4;\n"
		*conf += "\n"
	}
	if hasVIP6 {
		*conf += "ipv6 table drop6;\n"
		*conf += "\n"
	}
	if hasVIP4 {
		*conf += "protocol kernel {\n"
		*conf += "\tipv4 {\n"
		*conf += "\t\ttable drop4;\n"
		*conf += "\t\timport none;\n"
		*conf += "\t\texport all;\n"
		*conf += "\t};\n"
		*conf += "\tkernel table " + strconv.FormatInt(int64(fes.kernelTableId+1), 10) + ";\n"
		*conf += "}\n"
		*conf += "\n"
	}
	if hasVIP6 {
		*conf += "protocol kernel {\n"
		*conf += "\tipv6 {\n"
		*conf += "\t\ttable drop6;\n"
		*conf += "\t\timport none;\n"
		*conf += "\t\texport all;\n"
		*conf += "\t};\n"
		*conf += "\tkernel table " + strconv.FormatInt(int64(fes.kernelTableId+1), 10) + ";\n"
		*conf += "}\n"
		*conf += "\n"
	}
	if hasVIP4 {
		*conf += "protocol static DROP4 {\n"
		*conf += "\tipv4 { table drop4; preference 0; };\n"
		*conf += "\troute 0.0.0.0/0 blackhole {\n"
		*conf += "\t\tkrt_metric=4294967295;\n"
		*conf += "\t\tigp_metric=4294967295;\n"
		*conf += "\t};\n"
		*conf += "}\n"
		*conf += "\n"
	}
	if hasVIP6 {
		*conf += "protocol static DROP6 {\n"
		*conf += "\tipv6 { table drop6; preference 0; };\n"
		*conf += "\troute 0::/0 blackhole {\n"
		*conf += "\t\tkrt_metric=4294967295;\n"
		*conf += "\t\tigp_metric=4294967295;\n"
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
		return fmt.Errorf("%v; %v", err, stringOut)
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
			rule.Priority = rulePriorityVIP
			rule.Table = fes.kernelTableId
			rule.Src = utils.StrToIPNet(vip)

			logger.Info("Remove VIP rule", "rule", rule)
			if err := handler.RuleDel(rule); err != nil {
				// keep error printout even in case of ENOENT error, lack of rule could have caused traffic issues before
				logger.Error(err, "Remove VIP rule")
			}

			rule.Priority = rulePriorityVIP + 1
			rule.Table = fes.kernelTableId + 1
			logger.Info("Remove VIP rule", "rule", rule)
			if err := handler.RuleDel(rule); err != nil {
				// keep error printout even in case of ENOENT error, lack of rule could have caused traffic issues before
				logger.Error(err, "Remove VIP rule")
			}
		}

		for _, vip := range vipsAdded {
			rule := netlink.NewRule()
			rule.Priority = rulePriorityVIP
			rule.Table = fes.kernelTableId
			rule.Src = utils.StrToIPNet(vip)

			logger.Info("Add VIP rule", "rule", rule)
			if err := handler.RuleAdd(rule); err != nil {
				if !errors.Is(err, os.ErrExist) {
					logger.Error(err, "Add VIP rule")
					return err
				}
			}

			// Add a second rule with lower priority in order to catch patckets that haven't matched.
			// Should prevent misrouting.
			// Note: IP Rules could be of type blackhole, but the go package does not support this.
			rule.Priority = rulePriorityVIP + 1
			rule.Table = fes.kernelTableId + 1
			logger.Info("Add VIP rule", "rule", rule)
			if err := handler.RuleAdd(rule); err != nil {
				if !errors.Is(err, os.ErrExist) {
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
