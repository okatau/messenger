package main

import (
	"context"
	"fmt"
	"friends_service/internal/components"
	"friends_service/internal/handler"
	"friends_service/internal/middleware"
	"friends_service/pkg/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
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

	// TODO add TLS
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
		Handler: router,
		// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		// WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	if err := serv.ListenAndServe(); err != nil {
		components.Logger.Info(fmt.Sprintf("listening friends service on %d", cfg.ServerConfig.Port))
		log.Printf("server stopped: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	serv.Shutdown(shutdownCtx)
	components.Shutdown(shutdownCtx)
}
