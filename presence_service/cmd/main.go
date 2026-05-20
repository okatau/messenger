package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"presence_service/internal/components"
	"presence_service/internal/server"
	"syscall"

	"presence_service/pkg/config"
	pb "presence_service/pkg/pb"

	"google.golang.org/grpc"
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

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ServerConfig.GRPCPort))
	if err != nil {
		log.Fatal("grpc listen: %w", err)
	}
	comps.Logger.Info(fmt.Sprintf("listening grpc friends service on %d", cfg.ServerConfig.GRPCPort))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("grpc server: %w", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	grpcServer.GracefulStop()
	comps.Shutdown(shutdownCtx)
}
