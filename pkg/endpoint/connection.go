package endpoint

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type NSEConnectionFactory interface {
	NewNSEIPContext() (*networkservice.IPContext, error)
}
