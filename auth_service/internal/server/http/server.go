package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"auth_service/internal/service"
	"auth_service/pkg/config"
	"auth_service/pkg/service_logger"
)

type Server struct {
	srv    *http.Server
	logger *slog.Logger
}

func New(cfg config.ServerConfig, svc service.Auth, logger *slog.Logger) *Server {
	router := echo.New()
	router.Use(service_logger.LoggerMW(logger))
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
