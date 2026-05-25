package main

import (
	"context"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"presence_service/internal/components"
	"presence_service/internal/server"
	"presence_service/pkg/config"
	pb "presence_service/pkg/pb"
	"presence_service/pkg/service_logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load[components.Config]()

	initCtx, cancelInit := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelInit()

	comps := components.InitComponents(initCtx, cfg)

	grpcServer := grpc.NewServer()
	pb.RegisterPresenceServer(grpcServer, server.NewPresenceServer(comps.Svc))

	//nolint:noctx // net.Listen completes instantly; gRPC shutdown is handled via GracefulStop
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ServerConfig.GRPCPort))
	if err != nil {
		comps.Logger.Error("error net.Listen", service_logger.Err(err))
	}
	comps.Logger.Info(fmt.Sprintf("listening grpc friends service on %d", cfg.ServerConfig.GRPCPort))
	if err := grpcServer.Serve(lis); err != nil {
		comps.Logger.Error("error grpc listen", service_logger.Err(err))
	}

	grpcServer.GracefulStop()
	comps.Shutdown()
}
