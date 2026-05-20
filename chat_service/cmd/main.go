package main

import (
	"chat_service/internal/components"
	"chat_service/internal/server/httpserver"
	"chat_service/pkg/config"
	"chat_service/pkg/service_logger"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()

	hubCtx, hubCancel := context.WithCancel(ctx)
	defer hubCancel()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	comps := components.InitComponents(ctxTimeout, hubCtx, cfg)

	comps.Logger.Info(fmt.Sprintf("Running on %s environment", cfg.Env))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serv := httpserver.New(
		hubCtx,
		cfg.ServerConfig,
		cfg.OriginWhitelist,
		comps.Hub,
		comps.Logger,
		comps.TokenManager,
	)

	go func() {
		comps.Logger.Info(fmt.Sprintf("listening chat service on %d", cfg.ServerConfig.Port))
		if err := serv.Start(); err != nil {
			comps.Logger.Error("auth service stopped: %v", service_logger.Err(err))
		}
	}()

	<-quit
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	serv.Stop(shutdownCtx)
	comps.Shutdown(shutdownCtx)
}
