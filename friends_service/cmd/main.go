package main

import (
	"context"
	"fmt"
	"friends_service/internal/components"
	"friends_service/internal/handler"
	"friends_service/internal/middleware"
	"friends_service/pkg/config"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load[components.Config]()
	log.Println("friends config", cfg)
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, time.Second*10)
	defer cancelTimeout()

	components := components.InitComponents(ctxTimeout, cfg)

	auth := middleware.Auth(components.TokenManager)
	loggerMW := middleware.Logger(components.Logger)
	router := echo.New()
	router.Use(auth)
	router.Use(loggerMW)

	router.GET("", handler.GetFriendsList(components.Svc))
	router.GET("/search", handler.SearchUser(components.Svc)) // TODO semantics

	router.POST("/add", handler.SendFriendRequest(components.Svc))
	router.POST("/accept", handler.AcceptFriendRequest(components.Svc))
	router.POST("/decline", handler.DeclineFriendRequest(components.Svc))
	router.POST("/cancel", handler.CancelFriendRequest(components.Svc))

	router.DELETE("/:friendId", handler.RemoveFriend(components.Svc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-quit
		cancel()
	}()

	sc := echo.StartConfig{
		Address:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		GracefulTimeout: 10 * time.Second,
	}

	if err := sc.Start(ctx, router); err != nil {
		log.Printf("server stopped: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	components.Shutdown(shutdownCtx)
}
