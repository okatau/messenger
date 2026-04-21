package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chat_service/internal/components"
	"chat_service/internal/handler"
	"chat_service/internal/middleware"
	"chat_service/pkg/config"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	hubCtx, hubCancel := context.WithCancel(ctx)
	defer hubCancel()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelTimeout()

	cfg := config.Load[components.Config]()

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

	go func() {
		if err := router.Start(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	components.Shutdown(shutdownCtx)
}
