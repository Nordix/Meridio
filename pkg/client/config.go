package client

import (
	"errors"
	"net/url"
	"time"
)

// Config - configuration for cmd-nsmgr
type Config struct {
	Name             string        `default:"nsc" desc:"Name of Network Service Client"`
	ConnectTo        url.URL       `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to NSM" split_words:"true"`
	DialTimeout      time.Duration `default:"5s" desc:"timeout to dial NSMgr" split_words:"true"`
	RequestTimeout   time.Duration `default:"15s" desc:"timeout to request NSE" split_words:"true"`
	MaxTokenLifetime time.Duration `default:"24h" desc:"maximum lifetime of tokens" split_words:"true"`
	Labels           []string      `default:"" desc:"A list of client labels with format key1=val1,key2=val2, will be used a primary list for network services" split_words:"true"`
	Mechanism        string        `default:"kernel" desc:"Default Mechanism to use, supported values: kernel, vfio" split_words:"true"`
	NetworkServices  []url.URL     `default:"" desc:"A list of Network Service Requests" split_words:"true"`
}

// IsValid - check if configuration is valid
func (c *Config) IsValid() error {
	if len(c.NetworkServices) == 0 {
		return errors.New("no network services are specified")
	}
	if c.Name == "" {
		return errors.New("no client name specified")
	}
	if c.ConnectTo.String() == "" {
		return errors.New("no NSMGr ConnectTO URL are specified")
	}
	return nil
}

// NetworkServiceConfig - defines a network service request configuration
type NetworkServiceConfig struct {
	NetworkService string            `default:"" desc:"A name of network service" split_words:"true"`
	Path           []string          `default:"" desc:"An interfaceName or memif socket file name" split_words:"true"`
	Mechanism      string            `default:"" desc:"Mechanism used by client" split_words:"true"`
	Labels         map[string]string `default:"" desc:"A map of client labels" split_words:"true"`
}
