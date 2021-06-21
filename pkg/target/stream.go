package target

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/nsp"
)

type Stream struct {
	identifier         int
	networkServiceName string
	trench             string
	ips                []string
	config             *Config
}

func (s *Stream) Request() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	targetContext := map[string]string{
		"identifier": strconv.Itoa(s.identifier),
	}
	err = nspClient.Register(s.ips, targetContext)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}

func (s *Stream) Close() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	err = nspClient.Unregister(s.ips)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}

func (s *Stream) getNSPService() string {
	return fmt.Sprintf("%s.%s:%d", s.config.nspServiceName, s.trench, s.config.nspServicePort)
}

func NewStream(networkServiceName string, trench string, ips []string, config *Config) *Stream {
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	stream := &Stream{
		identifier:         identifier,
		ips:                ips,
		networkServiceName: networkServiceName,
		trench:             trench,
		config:             config,
	}
	return stream
}
