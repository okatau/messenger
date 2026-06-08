package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"presence_service/internal/components"
	grpcserver "presence_service/internal/server/grpc"
	"presence_service/pkg/config"
	"presence_service/pkg/pb"
	"presence_service/pkg/service_logger"
)

type Server struct {
	grpc  *grpc.Server
	comps *components.Components
	cfg   config.ServerConfig
}

func New(ctx context.Context) *Server {
	cfg := config.Load[components.Config]()

	initCtx, cancelInit := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelInit()

	comps := components.InitComponents(initCtx, cfg)

	grpcServer := grpc.NewServer()
	pb.RegisterPresenceServer(grpcServer, grpcserver.New(comps.Svc))

	return &Server{
		grpc:  grpcServer,
		comps: comps,
		cfg:   cfg.ServerConfig,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPCPort))
	if err != nil {
		s.comps.Logger.Error("error net.Listen", service_logger.Err(err))
		return err
	}
	s.comps.Logger.Info(fmt.Sprintf("listening grpc friends service on %d", s.cfg.GRPCPort))
	return s.grpc.Serve(lis)
}

func (s *Server) Stop() {
	s.grpc.GracefulStop()
	s.comps.Shutdown()
}
