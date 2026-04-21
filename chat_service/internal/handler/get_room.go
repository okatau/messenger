package handler

import (
	"net/http"

	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

func GetRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)

		rooms, err := hub.GetRoomsByUser(c.Request().Context(), userID)
		if err != nil {

			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, rooms)
	}
}
