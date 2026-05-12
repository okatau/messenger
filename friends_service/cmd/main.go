package main

import (
	"context"
	"errors"
	"fmt"
	"friends_service/internal/components"
	"friends_service/internal/handler"
	grpcHandler "friends_service/internal/handler/grpc"
	"friends_service/internal/middleware"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
	"friends_service/pkg/service_rate_limiter"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	pb "friends_service/pkg/friendspb"

	"github.com/labstack/echo/v5"
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

	auth := middleware.Auth(comps.TokenManager)
	loggerMW := service_logger.LoggerMW(comps.Logger)
	rl := func(limitRate int) echo.MiddlewareFunc {
		return service_rate_limiter.RateLimitByUser(comps.Limiter, comps.Logger, limitRate)
	}

	router := echo.New()
	router.Use(auth)
	router.Use(loggerMW)

	router.GET("", handler.GetFriendsList(comps.Svc))
	router.GET("/search", handler.SearchUser(comps.Svc), rl(cfg.Limits.SearchLimit))
	router.GET("/invites", handler.GetInvites(comps.Svc), rl(cfg.Limits.SearchLimit))
	router.GET("/search/friend", handler.SearchFriend(comps.Svc), rl(cfg.Limits.SearchLimit))

	router.POST("/add", handler.SendFriendRequest(comps.Svc), rl(cfg.Limits.AddLimit))
	router.POST("/accept", handler.AcceptFriendRequest(comps.Svc))
	router.POST("/decline", handler.DeclineFriendRequest(comps.Svc))
	router.POST("/cancel", handler.CancelFriendRequest(comps.Svc)) // no use for now

	router.DELETE("/:friendId", handler.RemoveFriend(comps.Svc))

	// TODO add TLS
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
		Handler: router,
	}

	grpcServer := grpc.NewServer()
	pb.RegisterFriendshipServer(grpcServer, &grpcHandler.GRPCServer{Svc: comps.Svc})

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		comps.Logger.Info(fmt.Sprintf("listening friends service on %d", cfg.ServerConfig.Port))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}

		comps.Shutdown(shutdownCtx)
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
