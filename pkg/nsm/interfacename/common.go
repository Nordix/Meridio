package interfacename

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
)

const MAX_INTERFACE_NAME_LENGTH = 16

type interfaceNameSetter struct {
	nameGenerator NameGenerator
	prefix        string
	maxLength     int
}

func (ins *interfaceNameSetter) SetInterfaceName(request *networkservice.NetworkServiceRequest) {
	if request == nil || request.GetConnection() == nil || request.GetConnection().GetMechanism() == nil {
		return
	}
	mechanism := request.GetConnection().GetMechanism()
	if mechanism.GetParameters() == nil {
		mechanism.Parameters = make(map[string]string)
	}
	mechanism.GetParameters()[common.InterfaceNameKey] = ins.nameGenerator.Generate(ins.prefix, ins.maxLength)
}

// NewInterfaceNameEndpoint -
func NewInterfaceNameSetter(prefix string, generator NameGenerator, maxLength int) *interfaceNameSetter {
	return &interfaceNameSetter{
		nameGenerator: generator,
		prefix:        prefix,
		maxLength:     maxLength,
	}
}
