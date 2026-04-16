package handler

import (
	"chat_service/internal/service"
	"net/http"

	"github.com/labstack/echo/v5"
)

func GetRoomActiveUsers(hub service.Hub) echo.HandlerFunc {
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
