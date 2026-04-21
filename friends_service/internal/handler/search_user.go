package handler

import (
	"net/http"

	"friends_service/internal/service"

	"github.com/labstack/echo/v5"
)

func SearchUser(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		username := c.QueryParam("username")
		cursor := c.QueryParam("cursor")
		if username == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "username query param is required")
		}

		users, err := svc.FindMatchingUsers(c.Request().Context(), username, cursor)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, users)
	}
}
