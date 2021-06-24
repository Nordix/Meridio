package main

import (
	"bufio"
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// FrontEndService -
func NewFrontEndService(config *Config) *FrontEndService {
	logrus.Infof("NewFrontEndService")
	frontEndService := &FrontEndService{
		vips:          config.VIPs,
		vipsWithRules: []string{},
		gateways:      config.Gateways,
		vrrps:         config.VRRPs,
		birdConfPath:  config.BirdConfigPath,
		birdConfFile:  config.BirdConfigPath + "/bird-fe-meridio.conf",
		kernelTableId: config.TableID,
		extInterface:  config.ExternalInterface,
		localAS:       config.LocalAS,
		remoteAS:      config.RemoteAS,
		localPortBGP:  config.BGPLocalPort,
		remotePortBGP: config.BGPRemotePort,
		holdTimeBGP:   config.BGPHoldTime,
		ecmp:          config.ECMP,
		bfd:           config.BFD,
		dropIfNoPeer:  config.DropIfNoPeer,
		logBird:       config.LogBird,
	}

	if len(frontEndService.vrrps) > 0 {
		// When using static default routes there's no need for blackhole routes...
		frontEndService.dropIfNoPeer = false
	}

	return frontEndService
}

// FrontEndService -
// TODO: maybe lock vips
type FrontEndService struct {
	vips          []string
	vipsWithRules []string
	gateways      []string
	vrrps         []string
	birdConfPath  string
	birdConfFile  string
	kernelTableId int
	extInterface  string
	localAS       string
	remoteAS      string
	localPortBGP  string
	remotePortBGP string
	holdTimeBGP   string
	ecmp          bool
	bfd           bool
	dropIfNoPeer  bool
	logBird       bool
}

// Start -
// Start BIRD with the generated config
func (fes *FrontEndService) Start(ctx context.Context) <-chan error {
	logrus.Infof("FrontEndService: Start")
	errCh := make(chan error, 1)
	go fes.start(ctx, errCh)
	return errCh
}

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
			select {
			case <-ctx.Done():
				logrus.Infof("FrontEndService: context closed, terminate log monitoring...")
			}
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
		for scanner.Scan() {
			if ok, _ := regexp.MatchString(`Error|<ERROR>|<BUG>|<FATAL>|<WARNING>`, scanner.Text()); ok {
				logrus.Warnf("[bird] %v", scanner.Text())
			} else if ok, _ := regexp.MatchString(`<INFO>|BGP session|Connected|Received:|Started|Neighbor|Startup delayed`, scanner.Text()); ok {
				logrus.Infof("[bird] %v", scanner.Text())
			} else {
				//logrus.Debugf("[bird] %v", scanner.Text())
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

// AddVipRules -
// Add src based routing rules for VIP addresses (pointing to the routing table BIRD shall sync to)
// TODO: save vips who have rules installed (they might get deleted later...)
func (fes *FrontEndService) AddVipRules(cancel context.CancelFunc) error {
	handler, err := netlink.NewHandle()
	if err != nil {
		logrus.Errorf("AddVipRules: failed open handler: %v", err)
		cancel()
		return err
	}
	defer handler.Delete()

	for _, vip := range fes.vips {
		rule := netlink.NewRule()
		rule.Priority = 100
		rule.Table = fes.kernelTableId
		rule.Src = strToIPNet(vip)

		logrus.Infof("AddVipRules: adding rule: %v", rule)
		if err := handler.RuleAdd(rule); err != nil {
			logrus.Errorf("AddVipRules: failed adding rule %v", err)
			cancel()
			return err
		}
		// save vips for whom we have installed ip rule, so that when the set
		// of vips changes, we could detect and follow up changes
		fes.vipsWithRules = append(fes.vipsWithRules, vip)
	}

	return nil
}

// RemoveVipRules -
// Remove all the previously installed VIP src routing rules
func (fes *FrontEndService) RemoveVipRules() error {
	if len(fes.vipsWithRules) > 0 {
		handler, err := netlink.NewHandle()
		if err != nil {
			logrus.Errorf("RemoveVipRules: failed open handler: %v", err)
			return err
		}
		defer handler.Delete()

		for _, vip := range fes.vipsWithRules {
			rule := netlink.NewRule()
			rule.Priority = 100
			rule.Table = fes.kernelTableId
			rule.Src = strToIPNet(vip)

			logrus.Infof("RemoveVipRules: rule: %v", rule)
			if err := handler.RuleDel(rule); err != nil {
				logrus.Warnf("RemoveVipRules: failed removing rule %v", err)
				//TODO: return with error unless error refers to ENOENT/ESRCH
			}
		}

		fes.vipsWithRules = nil
	}

	return nil
}

// Monitor -
// Check bgp prorocols' status by periodically querying birdc.
// Log changes in availablity/connectivity.
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
		extConnOK, init := true, true
		for {
			select {
			case <-ctx.Done():
				logrus.Infof("Monitor: shutting down")
				return
			case <-time.After(5 * time.Second): //timeout
			}

			arg := `"bgp*"`
			cmd := exec.CommandContext(ctx, lp, "show", "protocols", "all", arg)
			//cmd := exec.CommandContext(ctx, lp, "show", "protocols", arg)
			stdoutStderr, err := cmd.CombinedOutput()
			stringOut := string(stdoutStderr)
			if err != nil {
				logrus.Warnf("Monitor: %v: %v", err, stringOut)
				//Note: if birdc is not yet running, no need to bail out
				//linkCh <- "Failed to fetch protocol status"
			} else if strings.Contains(stringOut, "No protocols match") {
				if extConnOK {
					extConnOK = false
					logrus.Warnf("Monitor: %v", stringOut)
					//linkCh <- "No protocols match"
				}
			} else {
				// Note: It is assumed, that the set of gateways can not change on the fly
				scanner := bufio.NewScanner(strings.NewReader(stringOut))
				scanOK := true
				scanDetails := ""
				for scanner.Scan() {
					if ok, _ := regexp.MatchString(`bgp[0-9]+`, scanner.Text()); ok {
						scanDetails += scanner.Text()
						if !strings.Contains(scanner.Text(), "Established") {
							//logrus.Debugf("Monitor: (scanOK->false) %v", scanner.Text())
							scanOK = false
						}
					} else if strings.Contains(scanner.Text(), `Neighbor address`) {
						scanDetails += scanner.Text() + "\n"
					} else if ok, _ := regexp.MatchString(`BIRD|Name\s+Proto`, scanner.Text()); ok {
						scanDetails += scanner.Text() + "\n"
					}
				}

				if extConnOK && !scanOK {
					extConnOK = false
					if !init {
						logrus.Warnf("Monitor: %v", scanDetails)
					}
				} else if !scanOK {
					// Keep printing protocol information until link is down
					extConnOK = false
					if !init {
						logrus.Debugf("Monitor: %v", scanDetails)
					}
				} else if !extConnOK && scanOK {
					extConnOK = true
					if !init {
						logrus.Infof("Monitor: %v", scanDetails)
					}
				}
				// TODO: ugly
				once.Do(func() {
					logrus.Infof("Monitor: %v", scanDetails)
					init = false
				})
			}
		}
	}()
	return nil
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

// ReloadConfig -
// Configure BIRD config file
//
// prerequisite: BIRD must be running so that birdc could talk to it
func (fes *FrontEndService) ReloadConfig(ctx context.Context) error {
	lp, err := exec.LookPath("birdc")
	if err != nil {
		logrus.Fatalf("ReloadConfig: Birdc not found!")
		return err
	} else {
		arg := `"` + fes.birdConfFile + `"`
		cmd := exec.CommandContext(ctx, lp, "configure", arg)
		stdoutStderr, err := cmd.CombinedOutput()
		stringOut := string(stdoutStderr)
		if err != nil {
			logrus.Errorf("ReloadConfig: %v: %v", err, stringOut)
			return err
		} else if !strings.Contains(stringOut, "Reconfiguration in progress") && !strings.Contains(stringOut, "Reconfigured") {
			logrus.Errorf("ReloadConfig: %v", stringOut)
			return errors.New("ReloadConfig failed")
		} else {
			logrus.Debugf("ReloadConfig: %v", stringOut)
			return nil
		}
	}
}

// WriteConfig -
// Create BIRD config file
//
// Can be used both for the initial config and for later changes as well. (BIRD can
// reconfigure itself based by loading the new config file - refer to ReloadConfig())
func (fes *FrontEndService) WriteConfig(ctx context.Context, cancel context.CancelFunc) {
	file, err := os.Create(fes.birdConfFile)
	if err != nil {
		logrus.Fatalf("FrontEndService: failed to create %v, err: %v", fes.birdConfFile, err)
		cancel()
	}
	defer file.Close()

	//conf := "include \"bird-common.conf\";\n"
	//conf += "\n"
	conf := ""
	fes.WriteConfigBase(&conf)
	hasVIP4, hasVIP6 := fes.WriteConfigVips(&conf)
	if len(fes.vrrps) > 0 {
		fes.WriteConfigVRRPs(&conf, hasVIP4, hasVIP6)
	} else if fes.dropIfNoPeer {
		fes.WriteConfigDropIfNoPeer(&conf, hasVIP4, hasVIP6)
	}
	fes.WriteConfigKernel(&conf, hasVIP4, hasVIP6)
	fes.WriteConfigBGP(&conf)

	logrus.Infof("FrontEndService: config generated")
	logrus.Debugf("\n%v", conf)
	_, err = file.WriteString(conf)
	if err != nil {
		logrus.Fatalf("FrontEndService: failed to write %v, err: %v", fes.birdConfFile, err)
		cancel()
	}
}

func (fes *FrontEndService) WriteConfigBase(conf *string) {
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

// WriteConfigBGP -
// Create BGP proto part of the BIRD config for each gateway to connect with them
//
// BGP is restricted to the external interface.
// Only VIP related routes are announced to peer, and only default routes are accepted.
//
// Note: When VRRP IPs are configured, BGP sessions won't import any routes from external
// peers, as external routes are going to be taken care of by static default routes (VRRP IPs
// as next hops).
func (fes *FrontEndService) WriteConfigBGP(conf *string) {
	for _, gw := range fes.gateways {
		if isIPv6(gw) || isIPv4(gw) {
			ipv := ""
			if isIPv4(gw) {
				ipv += "\tipv4 {\n"
				if len(fes.vrrps) > 0 {
					ipv += "\t\timport none;\n"
				} else {
					ipv += "\t\timport filter default_v4;\n"
				}
				ipv += "\t\texport filter cluster_e_static;\n"
				ipv += "\t};\n"
			} else if isIPv6(gw) {
				ipv = "\tipv6 {\n"
				if len(fes.vrrps) > 0 {
					ipv += "\t\timport none;\n"
				} else {
					ipv += "\t\timport filter default_v6;\n"
				}
				ipv += "\t\texport filter cluster_e_static;\n"
				ipv += "\t};\n"
			}
			*conf += "protocol bgp from LINK {\n"
			*conf += "\tinterface \"" + fes.extInterface + "\";\n"
			*conf += "\tlocal port " + fes.localPortBGP + " as " + fes.localAS + ";\n"
			*conf += "\tneighbor " + strings.Split(gw, "/")[0] + " port " + fes.remotePortBGP + " as " + fes.remoteAS + ";\n"
			*conf += ipv
			*conf += "}\n"
			*conf += "\n"
		}
	}
}

// WriteConfigKernel -
// Create kernel proto part of the BIRD config
//
// Kernel proto is used to sync default routes learnt from BGP peer into
// local network stack (to the specified routing table).
// Note: No need to sync learnt default routes to stack, in case there are
// no VIPs configured for the particular IP family.
func (fes *FrontEndService) WriteConfigKernel(conf *string, hasVIP4 bool, hasVIP6 bool) {
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

// WriteConfigVips -
// Create static routes for VIP addresses in BIRD config
//
// VIP addresses are configured as static routes in BIRD. They are
// only advertised to BGP peers and not synced into local network stack.
func (fes *FrontEndService) WriteConfigVips(conf *string) (hasVIP4, hasVIP6 bool) {
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
		*conf += "protocol static VIP4 {\n"
		*conf += "\tipv4 { preference 110; };\n"
		*conf += v4
		*conf += "}\n"
		*conf += "\n"
	}

	if v6 != "" {
		hasVIP6 = true
		*conf += "protocol static VIP6 {\n"
		*conf += "\tipv6 { preference 110; };\n"
		*conf += v6
		*conf += "}\n"
		*conf += "\n"
	}
	return
}

// WriteConfigDropIfNoPeer -
// Create static default blackhole routes, that will be synced to the routing table
// used by VIP src routing rules.
//
// The aim is to drop packets from VIP src addresses when no external gateways are
// available, when configured accordingly. (So that the POD's default route installed
// for the prrimary network couldn't interfere.)
//
// Notes:
// - These routes are configured with the highest (linux) metric (-> lowest prio)
// - BIRD 2.0.7 has a strange behaviour that differs between IPv4 and IPv6:
//		- IPv4: - seemingly all the default routes are installed to the OS kernel routing
//                table including e.g. default blackhole routes with lower preference
//              - BIRD fails to remove all the BGP related routes when there's a BIRD
//                managed blackhole route for the same destination as well
//		- IPv6: default route with the highest preference gets installed to OS kernel routing table
func (fes *FrontEndService) WriteConfigDropIfNoPeer(conf *string, hasVIP4 bool, hasVIP6 bool) {
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
	return
}

// WriteConfigVRRPs -
// BIRD managed default static routes substituting other routing protocol related
// external routes.
func (fes *FrontEndService) WriteConfigVRRPs(conf *string, hasVIP4, hasVIP6 bool) {
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

func isIPv4(ip string) bool {
	return strings.Count(ip, ":") == 0
}

func isIPv6(ip string) bool {
	return strings.Count(ip, ":") >= 2
}

func strToIPNet(in string) *net.IPNet {
	if in == "" {
		return nil
	}
	ip, ipNet, err := net.ParseCIDR(in)
	if err != nil {
		return nil
	}
	ipNet.IP = ip
	return ipNet
}
