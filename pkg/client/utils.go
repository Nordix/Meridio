package client

import (
	"context"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"google.golang.org/grpc"
)

func expirationTimeIsNull(expirationTime *timestamp.Timestamp) bool {
	nullTImeStamp := &timestamp.Timestamp{
		Seconds: -1,
	}
	return expirationTime == nil || expirationTime.AsTime().Equal(nullTImeStamp.AsTime())
}

func newClient(ctx context.Context, name string, cc grpc.ClientConnInterface, additionalFunctionality ...networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return client.NewClient(context.Background(),
		cc,
		client.WithName(name),
		client.WithAuthorizeClient(authorize.NewClient()),
		client.WithAdditionalFunctionality(additionalFunctionality...))
}

func copyRequest(request *networkservice.NetworkServiceRequest) *networkservice.NetworkServiceRequest {
	if request == nil {
		return nil
	}

	newRequest := &networkservice.NetworkServiceRequest{}

	conn := request.GetConnection()
	if conn != nil {
		newRequest.Connection = &networkservice.Connection{
			Id:                         conn.Id,
			NetworkService:             conn.NetworkService,
			Labels:                     map[string]string{},
			NetworkServiceEndpointName: conn.NetworkServiceEndpointName,
			Payload:                    conn.Payload,
			Context: &networkservice.ConnectionContext{
				IpContext:       &networkservice.IPContext{},
				DnsContext:      &networkservice.DNSContext{},
				EthernetContext: &networkservice.EthernetContext{},
				ExtraContext:    map[string]string{},
			},
		}
		// copy Labels
		for key, value := range conn.Labels {
			newRequest.Connection.Labels[key] = value
		}
		if conn.GetContext() != nil {
			// copy ExtraContext
			for key, value := range conn.GetContext().ExtraContext {
				newRequest.Connection.Context.ExtraContext[key] = value
			}
		}
	}

	mechanismPreferences := request.GetMechanismPreferences()
	if mechanismPreferences != nil {
		newRequest.MechanismPreferences = []*networkservice.Mechanism{}
		for _, value := range mechanismPreferences {
			newMechanismPreference := &networkservice.Mechanism{
				Cls:        value.Cls,
				Type:       value.Type,
				Parameters: map[string]string{},
			}
			// copy Parameters
			for key, value := range value.Parameters {
				newMechanismPreference.Parameters[key] = value
			}
			newRequest.MechanismPreferences = append(newRequest.MechanismPreferences, newMechanismPreference)
		}
	}

	return newRequest
}
