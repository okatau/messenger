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
	"auth_service/internal/middleware"
	"auth_service/pkg/config"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	components := components.InitComponents(ctxTimeout, cfg)

	router := echo.New()

	router.Use(middleware.Logger(components.Logger))

	router.POST("/register", handler.Register(components.Svc))
	router.POST("/login", handler.Login(components.Svc))
	router.POST("/refresh", handler.Refresh(components.Svc))
	router.POST("/logout", handler.Logout(components.Svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// TODO add TLS
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
		Handler: router,
		// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		// WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	go func() {
		components.Logger.Info(fmt.Sprintf("listening auth service on %d", cfg.ServerConfig.Port))
		if err := serv.ListenAndServe(); err != nil {
			log.Printf("auth service stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	serv.Shutdown(shutdownCtx)
	components.Shutdown(shutdownCtx)
}
