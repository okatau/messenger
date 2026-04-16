package handler

import (
	"friends_service/internal/service"
	"net/http"

	"github.com/labstack/echo/v5"
)

func GetFriendsList(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)

		friends, err := svc.GetFriendsList(c.Request().Context(), userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, friends)
	}
}
