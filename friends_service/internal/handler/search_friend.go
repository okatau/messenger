package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"friends_service/internal/service"
)

func SearchFriend(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in friends_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		username := c.QueryParam("username")
		cursor := c.QueryParam("cursor")

		if username == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "username query param is required")
		}

		users, err := svc.SearchFriend(c.Request().Context(), userID, username, cursor)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, users)
	}
}
