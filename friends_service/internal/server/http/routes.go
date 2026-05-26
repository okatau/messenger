package httpserver

import (
	"friends_service/internal/handler"
	"friends_service/internal/service"

	"github.com/labstack/echo/v5"
)

func registreRoutes(router *echo.Echo, svc service.Friendship) {
	router.GET("", handler.GetFriendsList(svc))
	router.GET("/search", handler.SearchUser(svc))
	router.GET("/invites", handler.GetInvites(svc))
	router.GET("/search/friend", handler.SearchFriend(svc))

	router.POST("/add", handler.SendFriendRequest(svc))
	router.POST("/accept", handler.AcceptFriendRequest(svc))
	router.POST("/decline", handler.DeclineFriendRequest(svc))
	router.POST("/cancel", handler.CancelFriendRequest(svc)) // no use for now

	router.DELETE("/:friendId", handler.RemoveFriend(svc))
}
