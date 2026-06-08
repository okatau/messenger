package httpserver

import (
	"context"
	"time"

	"github.com/labstack/echo/v5"

	"chat_service/internal/handler"
	"chat_service/internal/middleware"
	"chat_service/internal/service"
	"chat_service/pkg/token_manager"
)

func registreRoutes(router *echo.Echo, svc service.Hub, timeout time.Duration) {
	authMW := middleware.ExtractUserID()
	timeoutMW := middleware.Timeout(timeout)

	g := router.Group("", timeoutMW, authMW)

	g.GET("", handler.GetRoom(svc))
	g.GET("/:roomId/users", handler.GetUsersByRoom(svc))
	g.GET("/:roomId/messages", handler.GetRoomHistory(svc))
	g.GET("/invite-avil", handler.GetInviteAvailability(svc))

	g.POST("", handler.CreateRoom(svc))
	g.POST("/dm", handler.CreateDM(svc))
	g.POST("/:roomId/invite", handler.InviteUser(svc))
	g.POST("/:roomId/leave", handler.LeaveRoom(svc))
	g.POST("/invite-avil", handler.ChangeInviteAvailability(svc))
}

func registerWS(ctx context.Context, router *echo.Echo, svc service.Hub, tm *token_manager.TokenManager, whitelist []string) {
	router.GET("/wss", handler.Connect(ctx, svc, tm, whitelist))
}
