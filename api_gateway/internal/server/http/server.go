package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"api_gateway/internal/components"
	"api_gateway/internal/middleware"
	"api_gateway/pkg/config"
	sl "api_gateway/pkg/service_logger"
)

type Server struct {
	srv   *http.Server
	comps *components.Components
}

func New() (*Server, error) {
	cfg := config.Load[components.Config]()
	comps := components.InitComponents(context.Background(), cfg)

	authMW := middleware.Auth(comps.TokenManager)
	rlIP := func(limitRate int) echo.MiddlewareFunc {
		return middleware.RateLimitByIP(comps.Limiter, comps.Logger, limitRate)
	}
	rlID := func(limitRate int) echo.MiddlewareFunc {
		return middleware.RateLimitByUser(comps.Limiter, comps.Logger, limitRate)
	}

	router := echo.New()
	router.Use(sl.LoggerMW(comps.Logger))

	groupV1 := router.Group("/api/v1")

	if err := registreAuthRoutes(groupV1, cfg.AuthAddr, rlIP, cfg.RateLimits.Al); err != nil {
		comps.Logger.Error("failed to register auth routes", sl.Err(err))
		return nil, err
	}
	if err := registerChatRoutes(groupV1, cfg.ChatAddr, rlID, cfg.RateLimits.Cl, authMW); err != nil {
		comps.Logger.Error("failed to register chat routes", sl.Err(err))
		return nil, err
	}
	if err := registerFriendsRoutes(groupV1, cfg.FriendsAddr, rlID, cfg.RateLimits.Fl, authMW); err != nil {
		comps.Logger.Error("failed to register friends routes", sl.Err(err))
		return nil, err
	}

	return &Server{
		srv: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.ServerConfig.Port),
			Handler:      router,
			ReadTimeout:  cfg.ServerConfig.ReadTimeout,
			WriteTimeout: cfg.ServerConfig.WriteTimeout,
		},
		comps: comps,
	}, nil
}

func (s *Server) Start() error {
	s.comps.Logger.Info(fmt.Sprintf("listening api gateway on %s", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.comps.Logger.Error("error shutting down server", sl.Err(err))
	}
	if err := s.comps.RedisCloser.Close(); err != nil {
		s.comps.Logger.Error("error closing redis", sl.Err(err))
	}
}
