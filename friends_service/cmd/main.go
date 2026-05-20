package main

import (
	"context"
	"errors"
	"fmt"
	"friends_service/internal/components"
	grpcserver "friends_service/internal/server/grpc"
	httpserver "friends_service/internal/server/http"
	"friends_service/pkg/config"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	pb "friends_service/pkg/friends_pb"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load[components.Config]()

	initCtx, cancelInit := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelInit()

	comps := components.InitComponents(initCtx, cfg)

	srvHttp := httpserver.New(cfg.ServerConfig, comps.Svc, comps.Logger)
	grpcServer := grpc.NewServer()
	pb.RegisterFriendshipServer(grpcServer, grpcserver.New(comps.Svc))

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		comps.Logger.Info(fmt.Sprintf("listening friends service on %d", cfg.ServerConfig.Port))
		if err := srvHttp.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ServerConfig.GRPCPort))
		if err != nil {
			return fmt.Errorf("grpc listen: %w", err)
		}
		comps.Logger.Info(fmt.Sprintf("listening grpc friends service on %d", cfg.ServerConfig.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("grpc server: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		grpcServer.GracefulStop()

		if err := srvHttp.Stop(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}

		comps.Shutdown(shutdownCtx)
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
