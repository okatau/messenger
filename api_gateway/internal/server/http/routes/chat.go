package routes

import (
	"net/http/httputil"

	"github.com/labstack/echo/v5"

	"api_gateway/internal/components"
)

func InitChatEndpoints(
	proxy *httputil.ReverseProxy,
	chat *echo.Group,
	limitFn func(limit int) echo.MiddlewareFunc,
	limits components.ChatLimits,
	redirect func(proxy *httputil.ReverseProxy) echo.HandlerFunc,
	auth echo.MiddlewareFunc,
) {
	chat.Use(auth)

	chat.GET("", redirect(proxy))
	chat.GET("/:roomId/users", redirect(proxy))
	chat.GET("/:roomId/messages", redirect(proxy), limitFn(limits.MessagesLimit))
	chat.GET("/invite-avil", redirect(proxy))

	chat.POST("", redirect(proxy), limitFn(limits.CreateRoomLimit))
	chat.POST("/dm", redirect(proxy), limitFn(limits.CreateRoomLimit))
	chat.POST("/:roomId/invite", redirect(proxy), limitFn(limits.InviteLimit))
	chat.POST("/:roomId/leave", redirect(proxy))
	chat.POST("/invite-avil", redirect(proxy))
}
