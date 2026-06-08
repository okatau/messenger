package routes

import (
	"net/http/httputil"

	"github.com/labstack/echo/v5"

	"api_gateway/internal/components"
)

func InitFriendsEndpoints(
	proxy *httputil.ReverseProxy,
	friends *echo.Group,
	limitFn func(limit int) echo.MiddlewareFunc,
	limits components.FriendsLimits,
	redirect func(proxy *httputil.ReverseProxy) echo.HandlerFunc,
	auth echo.MiddlewareFunc,
) {
	friends.Use(auth)

	friends.GET("", redirect(proxy)) // TODO rename list?
	friends.GET("/search", redirect(proxy), limitFn(limits.SearchLimit))
	friends.GET("/invites", redirect(proxy), limitFn(limits.SearchLimit))
	friends.GET("/search/friend", redirect(proxy), limitFn(limits.SearchLimit))

	friends.POST("/add", redirect(proxy), limitFn(limits.AddLimit))
	friends.POST("/accept", redirect(proxy))
	friends.POST("/decline", redirect(proxy))
	friends.POST("/cancel", redirect(proxy)) // no use for now

	friends.DELETE("/:friendId", redirect(proxy))
}
