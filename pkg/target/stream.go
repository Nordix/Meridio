package target

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nordix/meridio/pkg/nsp"
)

type Stream struct {
	name       string
	identifier int
	conduit    *Conduit
}

func (s *Stream) Request() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	targetContext := map[string]string{
		"identifier": strconv.Itoa(s.identifier),
	}
	err = nspClient.Register(s.conduit.ips, targetContext)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}

func (s *Stream) Delete() error {
	nspClient, err := nsp.NewNetworkServicePlateformClient(s.getNSPService())
	if err != nil {
		return err
	}
	err = nspClient.Unregister(s.conduit.ips)
	if err != nil {
		return err
	}
	return nspClient.Delete()
}

func (s *Stream) getNSPService() string {
	return fmt.Sprintf("%s-%s.%s:%d", s.GetConfig().nspServiceName, s.GetTrenchName(), s.GetNamespace(), s.GetConfig().nspServicePort)
}

func (s *Stream) GetName() string {
	return s.name
}

func (s *Stream) GetTrenchName() string {
	return s.conduit.GetTrenchName()
}

func (s *Stream) GetConduitName() string {
	return s.conduit.GetName()
}

func (s *Stream) GetNamespace() string {
	return s.conduit.GetNamespace()
}

func (s *Stream) GetConfig() *Config {
	return s.conduit.GetConfig()
}

func NewStream(name string, conduit *Conduit) *Stream {
	hostname, _ := os.Hostname()
	identifier := Hash(hostname, 100)
	stream := &Stream{
		name:       name,
		identifier: identifier,
		conduit:    conduit,
	}
	return stream
}
