package main

import (
	"errors"
	"net/url"
	"time"
)

// Config for the proxy
type Config struct {
	Name                string        `default:"proxy" desc:"Pod Name"`
	ServiceName         string        `default:"proxy" desc:"Name of the Network Service" split_words:"true"`
	ConnectTo           url.URL       `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to NSM" split_words:"true"`
	DialTimeout         time.Duration `default:"5s" desc:"timeout to dial NSMgr" split_words:"true"`
	RequestTimeout      time.Duration `default:"15s" desc:"timeout to request NSE" split_words:"true"`
	MaxTokenLifetime    time.Duration `default:"24h" desc:"maximum lifetime of tokens" split_words:"true"`
	VIPs                []string      `default:"20.0.0.1/32" desc:"Virtual IP address"`
	SubnetPools         []string      `default:"169.255.0.0/16" desc:"SubnetPool from which the proxy subnet will be allocated" split_words:"true"`
	SubnetPrefixLengths []int         `default:"24" desc:"Subnet prefix length which will be allocated" split_words:"true"`
	IPAMService         string        `default:"ipam-service:7777" desc:"IP (or domain) and port of the IPAM Service" split_words:"true"`
	Host                string        `default:"" desc:"Host name the proxy is running on" split_words:"true"`
	NetworkServiceName  string        `default:"load-balancer" desc:"Name of the network service the proxy request the connection" split_words:"true"`
	NSPService          string        `default:"nsp-service:7778" desc:"IP (or domain) and port of the NSP Service" split_words:"true"`
	Namespace           string        `default:"default" desc:"Namespace the pod is running on" split_words:"true"`
	ConfigMapName       string        `default:"meridio-configuration" desc:"Name of the ConfigMap containing the configuration" split_words:"true"`
}

// IsValid checks if the configuration is valid
func (c *Config) IsValid() error {
	if c.ConnectTo.String() == "" {
		return errors.New("no NSMGr ConnectTO URL are specified")
	}
	return nil
}
