package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"auth_service/internal/components"
	"auth_service/internal/handler"
	"auth_service/pkg/config"
	"auth_service/pkg/service_logger"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	comps := components.InitComponents(ctxTimeout, cfg)

	router := echo.New()

	router.Use(service_logger.LoggerMW(comps.Logger))

	router.POST("/register", handler.Register(comps.Svc))
	router.POST("/login", handler.Login(comps.Svc))
	router.POST("/refresh", handler.Refresh(comps.Svc))
	router.POST("/logout", handler.Logout(comps.Svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// TODO add TLS
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
		Handler: router,
		// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		// WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	comps.Logger.Info(fmt.Sprintf("Running on %s environment", cfg.Env))

	go func() {
		comps.Logger.Info(fmt.Sprintf("listening auth service on %d", cfg.ServerConfig.Port))
		if err := serv.ListenAndServe(); err != nil {
			log.Printf("auth service stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	serv.Shutdown(shutdownCtx)
	comps.Shutdown(shutdownCtx)
}
