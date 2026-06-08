package handler

import (
	"net/http"

	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

func GetRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		ctx := c.Request().Context()
		rooms, err := hub.GetRoomsByUser(ctx, userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, rooms)
	}
}
