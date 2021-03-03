package client

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type NSCConnectionFactory interface {
	NewNSCIPContext() (*networkservice.IPContext, error)
}
