package target

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
)

type Config struct {
	nspServiceName string
	nspServicePort int
	netUtils       networking.Utils
	nsmConfig      *nsm.Config
}

func NewConfig(nspServiceName string, nspServicePort int, netUtils networking.Utils, nsmConfig *nsm.Config) *Config {
	config := &Config{
		nspServiceName: nspServiceName,
		nspServicePort: nspServicePort,
		netUtils:       netUtils,
		nsmConfig:      nsmConfig,
	}
	return config
}
