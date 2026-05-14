package main

import (
	"api_gateway/internal/components"
	"api_gateway/internal/handlers"
	"api_gateway/internal/middleware"
	rate_limiter "api_gateway/internal/service"
	"api_gateway/pkg/config"
	"api_gateway/pkg/service_logger"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	cfg := config.Load[components.Config]()

	comps := components.InitComponents(context.Background(), cfg)

	authMW := middleware.Auth(comps.TokenManager)

	rlIP := func(limitRate int) echo.MiddlewareFunc {
		return rate_limiter.RateLimitByIP(comps.Limiter, comps.Logger, limitRate)
	}
	rlID := func(limitRate int) echo.MiddlewareFunc {
		return rate_limiter.RateLimitByUser(comps.Limiter, comps.Logger, limitRate)
	}

	router := echo.New()
	router.Use(service_logger.LoggerMW(comps.Logger))

	rv1 := router.Group("/api/v1")

	auth := rv1.Group("/auth")
	chat := rv1.Group("/rooms")
	friends := rv1.Group("/friends")

	handlers.InitAuthEndpoints(auth, cfg.AuthAddr, cfg.RateLimits.Al, rlIP)
	handlers.InitChatEndpoints(chat, cfg.ChatAddr, cfg.RateLimits.Cl, rlID, authMW)
	handlers.InitFriendsEndpoints(friends, cfg.FriendsAddr, cfg.RateLimits.Fl, rlID, authMW)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
		Handler: router,
		// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
		// WriteTimeout: cfg.ServerConfig.WriteTimeout,
	}

	go func() {
		comps.Logger.Info(fmt.Sprintf("listening api gateway service on %d", cfg.ServerConfig.Port))
		if err := serv.ListenAndServe(); err != nil {
			log.Printf("api gateway service stopped: %v", err)
		}
	}()

	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer shutdownCancel()

	if err := serv.Shutdown(shutdownCtx); err != nil {
		comps.Logger.Error("error shutting down server", service_logger.Err(err))
	}
}
