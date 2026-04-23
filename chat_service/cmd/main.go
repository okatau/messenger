package main

import (
	"chat_service/internal/components"
	"chat_service/internal/handler"
	"chat_service/internal/middleware"
	"chat_service/pkg/config"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	hubCtx, hubCancel := context.WithCancel(ctx)
	defer hubCancel()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()

	components := components.InitComponents(ctxTimeout, hubCtx, cfg)

	auth := middleware.Auth(components.TokenManager)

	router := echo.New()
	router.Use(middleware.Logger(components.Logger))

	router.GET("/wss", handler.Connect(components.Hub, components.TokenManager, hubCtx))

	router.GET("", handler.GetRoom(components.Hub), auth)
	router.GET("/:roomId/active", handler.GetActiveUsersByRoom(components.Hub), auth)
	router.GET("/:roomId/messages", handler.GetRoomHistory(components.Hub), auth)
	router.POST("", handler.CreateRoom(components.Hub), auth)
	router.POST("/:roomId/invite", handler.InviteUser(components.Hub), auth)
	router.POST("/:roomId/leave", handler.LeaveRoom(components.Hub), auth)

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
		components.Logger.Info(fmt.Sprintf("listening chat service on %d", cfg.ServerConfig.Port))
		if err := serv.ListenAndServe(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	<-quit
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	serv.Shutdown(shutdownCtx)
	components.Shutdown(shutdownCtx)
}
