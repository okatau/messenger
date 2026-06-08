package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"friends_service/internal/components"
	grpcserver "friends_service/internal/server/grpc"
	httpserver "friends_service/internal/server/http"
	"friends_service/pkg/config"
	pb "friends_service/pkg/friends_pb"
	sl "friends_service/pkg/service_logger"
)

type Server struct {
	http  *httpserver.Server
	grpc  *grpc.Server
	comps *components.Components
	cfg   *config.ServerConfig
}

func New(ctx context.Context) *Server {
	cfg := config.Load[components.Config]()

	initCtx, cancelInit := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelInit()

	comps := components.InitComponents(initCtx, cfg)

	httpSrv := httpserver.New(cfg.ServerConfig, comps.Svc, comps.Logger)
	grpcSrv := grpc.NewServer()
	pb.RegisterFriendshipServer(grpcSrv, grpcserver.New(comps.Svc))

	return &Server{
		http:  httpSrv,
		grpc:  grpcSrv,
		comps: comps,
	}
}

func (s *Server) Start(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		if err := s.http.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.comps.Logger.Error("http server", sl.Err(err))
			return err
		}
		return nil
	})

	g.Go(func() error {
		//nolint:noctx // net.Listen completes instantly; gRPC shutdown is handled via GracefulStop
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPCPort))
		if err != nil {
			s.comps.Logger.Error("grpc listen error", sl.Err(err))
			return err
		}
		s.comps.Logger.Info("listening grpc friends service on", slog.Int("port", s.cfg.GRPCPort))
		if err := s.grpc.Serve(lis); err != nil {
			s.comps.Logger.Error("grpc server", sl.Err(err))
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()

		s.grpc.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.http.Stop(shutdownCtx); err != nil {
			s.comps.Logger.Error("http shutdown", sl.Err(err))
			return err
		}

		s.comps.Shutdown()
		return nil
	})

	return g.Wait()
}
