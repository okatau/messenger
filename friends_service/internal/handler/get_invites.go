package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"friends_service/internal/service"
)

func GetInvites(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in friends_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		invites, err := svc.GetInvites(c.Request().Context(), userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error reading invites")
		}

		return c.JSON(http.StatusOK, invites)
	}
}
