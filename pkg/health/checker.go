package health

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Checker struct {
	listener net.Listener
	server   *grpc.Server
	port     int
}

func (c *Checker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: c.getStatus(),
	}, nil
}

func (c *Checker) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return server.Send(&grpc_health_v1.HealthCheckResponse{
		Status: c.getStatus(),
	})
}

func (s *Checker) getStatus() grpc_health_v1.HealthCheckResponse_ServingStatus {
	return grpc_health_v1.HealthCheckResponse_SERVING
}

func (c *Checker) Start() error {
	return c.server.Serve(c.listener)
}

func NewChecker(port int) (*Checker, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf("[::]:%s", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()

	checker := &Checker{
		listener: lis,
		server:   s,
		port:     port,
	}

	grpc_health_v1.RegisterHealthServer(s, checker)

	return checker, nil
}
