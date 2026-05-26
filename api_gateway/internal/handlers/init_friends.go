package handlers

import (
	"log"

	"github.com/labstack/echo/v5"

	"api_gateway/internal/components"
)

func InitFriendsEndpoints(
	friends *echo.Group,
	targetURL string,
	fl components.FriendsLimits,
	rl func(limit int) echo.MiddlewareFunc,
	auth echo.MiddlewareFunc,
) {
	proxy, err := createProxy(targetURL, "/api/v1/friends")
	if err != nil {
		log.Fatal(err)
	}
	friends.Use(auth)

	friends.GET("", redirectTo(proxy)) // TODO rename list?
	friends.GET("/search", redirectTo(proxy), rl(fl.SearchLimit))
	friends.GET("/invites", redirectTo(proxy), rl(fl.SearchLimit))
	friends.GET("/search/friend", redirectTo(proxy), rl(fl.SearchLimit))

	friends.POST("/add", redirectTo(proxy), rl(fl.AddLimit))
	friends.POST("/accept", redirectTo(proxy))
	friends.POST("/decline", redirectTo(proxy))
	friends.POST("/cancel", redirectTo(proxy)) // no use for now

	friends.DELETE("/:friendId", redirectTo(proxy))
}
