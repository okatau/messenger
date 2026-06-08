package handler

import (
	"net/http"

	"chat_service/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func GetUsersByRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		roomID := c.Param("roomId")
		rID, err := uuid.Parse(roomID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
		}

		users, err := hub.GetRoomClients(c.Request().Context(), rID.String())
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, users)
	}
}
