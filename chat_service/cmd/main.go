package main

import (
	"chat_service/internal/components"
	"chat_service/internal/handler"
	"chat_service/internal/middleware"
	"chat_service/pkg/config"
	"chat_service/pkg/service_logger"
	"chat_service/pkg/service_rate_limiter"
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

	comps := components.InitComponents(ctxTimeout, hubCtx, cfg)

	auth := middleware.Auth(comps.TokenManager)
	rl := func(limitRate int) echo.MiddlewareFunc {
		return service_rate_limiter.RateLimitByUser(comps.Limiter, comps.Logger, limitRate)
	}

	router := echo.New()
	router.Use(service_logger.LoggerMW(comps.Logger))

	router.GET("/wss", handler.Connect(comps.Hub, comps.TokenManager, hubCtx, cfg.OriginWhitelist))

	router.GET("", handler.GetRoom(comps.Hub), auth)
	router.GET("/:roomId/users", handler.GetUsersByRoom(comps.Hub), auth)
	router.GET("/:roomId/messages", handler.GetRoomHistory(comps.Hub), auth, rl(cfg.Limits.MessagesLimit))
	router.POST("", handler.CreateRoom(comps.Hub), auth, rl(cfg.Limits.CreateRoomLimit))
	router.POST("/:roomId/invite", handler.InviteUser(comps.Hub), auth, rl(cfg.Limits.InviteLimit))
	router.POST("/:roomId/leave", handler.LeaveRoom(comps.Hub), auth)

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
		comps.Logger.Info(fmt.Sprintf("listening chat service on %d", cfg.ServerConfig.Port))
		if err := serv.ListenAndServe(); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	<-quit
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer shutdownCancel()

	serv.Shutdown(shutdownCtx)
	comps.Shutdown(shutdownCtx)
}
