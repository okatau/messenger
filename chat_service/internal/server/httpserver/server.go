package httpserver

import (
	"chat_service/internal/service"
	"chat_service/pkg/config"
	"chat_service/pkg/service_logger"
	"chat_service/pkg/token_manager"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

type Server struct {
	serv   *http.Server
	logger *slog.Logger
}

func New(
	ctx context.Context,
	cfg config.HTTPConfig,
	whitelist []string,
	svc service.Hub,
	logger *slog.Logger,
	tm *token_manager.TokenManager,
) *Server {
	router := echo.New()
	router.Use(service_logger.LoggerMW(logger))

	registreRoutes(router, svc)
	registerWS(ctx, router, svc, tm, whitelist)

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
