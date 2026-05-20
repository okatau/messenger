package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"auth_service/internal/components"
	httpserver "auth_service/internal/server/http"
	"auth_service/pkg/config"
	"auth_service/pkg/service_logger"
)

func main() {
	ctx := context.Background()
	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	comps := components.InitComponents(ctxTimeout, cfg)

	comps.Logger.Info(fmt.Sprintf("Running on %s environment", cfg.Env))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serv := httpserver.New(cfg.ServerConfig, comps.Svc, comps.Logger)
	go func() {
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
