package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"friends_service/internal/middleware"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
)

type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

func New(
	cfg config.ServerConfig,
	svc service.Friendship,
	logger *slog.Logger,
) *Server {
	router := echo.New()
	router.Use(service_logger.LoggerMW(logger))
	router.Use(middleware.ExtractUserID())

	registreRoutes(router, svc)

	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
			// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
			// WriteTimeout: cfg.ServerConfig.WriteTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info(fmt.Sprintf("listening auth service on %s", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
