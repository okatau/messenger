package httpserver

import (
	"context"
	"fmt"
	"friends_service/internal/middleware"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

type Server struct {
	serv   *http.Server
	logger *slog.Logger
}

func New(
	cfg config.HTTPConfig,
	svc service.Friendship,
	logger *slog.Logger,
) *Server {
	router := echo.New()
	router.Use(service_logger.LoggerMW(logger))
	router.Use(middleware.ExtractUserID())

	registreRoutes(router, svc)

	return &Server{
		serv: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Port),
			Handler: router,
			// ReadTimeout:  cfg.ServerConfig.ReadTimeout,
			// WriteTimeout: cfg.ServerConfig.WriteTimeout,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info(fmt.Sprintf("listening auth service on %s", s.serv.Addr))
	return s.serv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.serv.Shutdown(ctx)
}
