package handler

import (
	"net/http"

	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

func GetActiveUsersByRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		roomID := c.Param("roomId")
		if roomID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
		}

		users := hub.GetRoomClients(roomID)

		type resType struct {
			Username string `json:"username"`
		}
		res := make([]resType, len(users))

		for i, u := range users {
			res[i] = resType{Username: u}
		}

		return c.JSON(http.StatusOK, res)
	}
}
