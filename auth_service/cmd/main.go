package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth_service/internal/components"
	"auth_service/internal/handler"
	"auth_service/internal/middleware"
	"auth_service/pkg/config"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, time.Second*10)
	defer cancelTimeout()
	components := components.InitComponents(ctxTimeout, cfg)

	router := echo.New()

	router.Use(middleware.Logger(components.Logger))

	router.POST("/register", handler.Register(components.Svc))
	router.POST("/login", handler.Login(components.Svc))
	router.POST("/refresh", handler.Refresh(components.Svc))
	router.POST("/logout", handler.Logout(components.Svc))

	components.Logger.Info(fmt.Sprintf("listening auth service on %d", cfg.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := router.Start(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)); err != nil {
			log.Printf("auth service stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	components.Shutdown(shutdownCtx)
}
