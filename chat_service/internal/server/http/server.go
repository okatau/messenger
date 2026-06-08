package httpserver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"chat_service/internal/components"
	"chat_service/pkg/config"
	sl "chat_service/pkg/service_logger"
)

type Server struct {
	srv   *http.Server
	comps *components.Components
}

func New(hubCtx context.Context) *Server {
	ctx := context.Background()
	cfg := config.Load[components.Config]()

	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, cfg.ServerConfig.ShutdownTimeout)
	defer cancelTimeout()
	comps := components.InitComponents(ctxTimeout, hubCtx, cfg)

	router := echo.New()
	router.Use(sl.LoggerMW(comps.Logger))

	registreRoutes(router, comps.Svc, cfg.ServerConfig.ReadTimeout)             //nolint:contextcheck // no need extra context
	registerWS(ctx, router, comps.Svc, comps.TokenManager, cfg.OriginWhitelist) //nolint:contextcheck // no need extra context

	return &Server{
		srv: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.ServerConfig.Port),
			Handler: router,
		},
		comps: comps,
	}
}

func (s *Server) Start() error {
	s.comps.Logger.Info(fmt.Sprintf("listening chat service on %s", s.srv.Addr))
	return s.srv.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.comps.Logger.Error("error shutting down server")
	}
	s.comps.Shutdown()
}
