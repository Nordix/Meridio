package ipcontext

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/nordix/meridio/pkg/networking"
)

type ipContextSetter interface {
	SetIPContext(conn *networkservice.Connection, interfaceType networking.InterfaceType) error
}
