package ipam

import (
	"context"
	"fmt"

	ipamAPI "github.com/nordix/meridio/api/ipam"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"
)

type IpamClient struct {
	ipamServiceClient ipamAPI.IpamServiceClient
}

func (ic *IpamClient) AllocateSubnet(subnetPool string, prefixLength int) (string, error) {
	subnetPoolAddr, err := netlink.ParseAddr(subnetPool)
	if err != nil {
		return "", err
	}
	subnetPoolPrefixLength, _ := subnetPoolAddr.Mask.Size()
	subnetRequest := &ipamAPI.SubnetRequest{
		SubnetPool: &ipamAPI.Subnet{
			Address:      subnetPoolAddr.IP.String(),
			PrefixLength: int32(subnetPoolPrefixLength),
		},
		PrefixLength: int32(prefixLength),
	}
	allocatedSubnet, err := ic.ipamServiceClient.Allocate(context.Background(), subnetRequest)
	if err != nil {
		return "", err
	}
	allocatedSubnetCIDR := fmt.Sprintf("%s/%d", allocatedSubnet.Address, allocatedSubnet.PrefixLength)
	return allocatedSubnetCIDR, nil
}

func (ic *IpamClient) connect(ipamServiceIPPort string) error {
	conn, err := grpc.Dial(ipamServiceIPPort, grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),
		))
	if err != nil {
		return nil
	}

	ic.ipamServiceClient = ipamAPI.NewIpamServiceClient(conn)
	return nil
}

func NewIpamClient(ipamServiceIPPort string) (*IpamClient, error) {
	ipamClient := &IpamClient{}
	err := ipamClient.connect(ipamServiceIPPort)
	if err != nil {
		return nil, err
	}
	return ipamClient, nil
}
