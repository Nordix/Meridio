package target

import (
	"github.com/nordix/meridio/pkg/networking"
	"github.com/nordix/meridio/pkg/nsm"
)

type Config struct {
	configMapName  string
	nspServiceName string
	nspServicePort int
	netUtils       networking.Utils
	nsmConfig      *nsm.Config
	apiClient      *nsm.APIClient
}

func NewConfig(configMapName string, nspServiceName string, nspServicePort int, netUtils networking.Utils, nsmConfig *nsm.Config) *Config {
	config := &Config{
		configMapName:  configMapName,
		nspServiceName: nspServiceName,
		nspServicePort: nspServicePort,
		netUtils:       netUtils,
		nsmConfig:      nsmConfig,
	}
	return config
}
