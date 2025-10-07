package logutils

import (
	ambassadorAPI "github.com/nordix/meridio/api/ambassador/v1"
	lbAPI "github.com/nordix/meridio/api/loadbalancer/v1"
	nspAPI "github.com/nordix/meridio/api/nsp/v1"
	"github.com/nordix/meridio/pkg/loadbalancer/types"
	"github.com/nordix/meridio/pkg/networking"
	"google.golang.org/grpc/connectivity"
)

// LogValue represents a key-value pair for structured logging
type LogValue struct {
	Key   string
	Value interface{}
}

// ToKV converts log values to a slice of alternating keys and values
// This allows using the helpers with any structured logger
func ToKV(values ...LogValue) []interface{} {
	kvs := make([]interface{}, 0, len(values)*2)
	for _, v := range values {
		kvs = append(kvs, v.Key, v.Value)
	}
	return kvs
}

// ConnectionIDValue returns a connection ID as a structured log value
func ConnectionIDValue(id string) LogValue {
	return LogValue{
		Key:   ConnectionID,
		Value: id,
	}
}

// InterfaceTypeValue returns an interface type as a structured log value
func InterfaceTypeValue(ifaceType networking.InterfaceType) LogValue {
	return LogValue{
		Key:   InterfaceType,
		Value: ifaceType,
	}
}

// ErrorValue returns an error as a structured log value
func ErrorValue(err error) LogValue {
	return LogValue{
		Key:   Error,
		Value: err,
	}
}

// InterfaceObject returns an interface object as a structured log value
func InterfaceObjectValue(obj networking.Iface) LogValue {
	return LogValue{
		Key:   InterfaceObject,
		Value: obj,
	}
}

// InterfaceName returns interface name as a structured log value
func InterfaceNameValue(name string) LogValue {
	return LogValue{
		Key:   InterfaceName,
		Value: name,
	}
}

// PreferredInterfaceValue returns a preferred interface name as a structured log value
func PreferredInterfaceNameValue(name string) LogValue {
	return LogValue{
		Key:   PreferredInterface,
		Value: name,
	}
}

// InterfaceIndex returns interface index as a structured log value
func InterfaceIndexValue(index int) LogValue {
	return LogValue{
		Key:   InterfaceIndex,
		Value: index,
	}
}

// FunctionValue returns a function name as a structured log value
func FunctionValue(funcName string) LogValue {
	return LogValue{
		Key:   Function,
		Value: funcName,
	}
}

// ConduitValue returns a conduit object as a structured log value
func ConduitValue(conduit *nspAPI.Conduit) LogValue {
	return LogValue{
		Key:   Conduit,
		Value: conduit,
	}
}

// VIPsValue returns a list of VIPs as a structured log value
func VipsValue(vips []*nspAPI.Vip) LogValue {
	return LogValue{
		Key:   Vips,
		Value: vips,
	}
}

// Target returns a target as a structured log value
func LBTargetValue(target types.Target) LogValue {
	return LogValue{
		Key:   LBTarget,
		Value: target,
	}
}

// TargetsValue returns a list of targets as a structured log value
func TargetsValue(targets []*nspAPI.Target) LogValue {
	return LogValue{
		Key:   Targets,
		Value: targets,
	}
}

// TargetValue returns a target as a structured log value
func LbApiTargetValue(target *lbAPI.Target) LogValue {
	return LogValue{
		Key:   LbAPiTarget,
		Value: target,
	}
}

// TargetValue returns a target as a structured log value
func TargetValue(target *nspAPI.Target) LogValue {
	return LogValue{
		Key:   Target,
		Value: target,
	}
}

// StreamValue returns a stream as a structured log value
func StreamValue(stream *ambassadorAPI.Stream) LogValue {
	return LogValue{
		Key:   Stream,
		Value: stream,
	}
}

// IdentifierValue returns an identifier as a structured log value
func IdentifierValue(id int) LogValue {
	return LogValue{
		Key:   Identifier,
		Value: id,
	}
}

// HealthServiceValue returns a health service name as a structured log value
func HealthServiceValue(service string) LogValue {
	return LogValue{
		Key:   HealthService,
		Value: service,
	}
}

// ConnectionStateValue returns a connection state as a structured log value
func ConnectionStateValue(state connectivity.State) LogValue {
	return LogValue{
		Key:   ConnectionState,
		Value: state,
	}
}

// ClientTargetValue returns a client target as a structured log value
func ClientTargetValue(target string) LogValue {
	return LogValue{
		Key:   ClientTarget,
		Value: target,
	}
}
