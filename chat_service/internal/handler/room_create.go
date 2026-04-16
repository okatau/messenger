package handler

import (
	"chat_service/internal/service"
	"net/http"

	"github.com/labstack/echo/v5"
)

func CreateRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)

		var req struct {
			Name string `json:"name"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		if req.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room name")
		}

		room, err := hub.CreateRoom(c.Request().Context(), req.Name, userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusCreated, room)
	}
}
