package handler

import (
	"chat_service/internal/domain"
	"chat_service/internal/service"
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

func LeaveRoom(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		roomID := c.Param("roomId")

		if roomID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid roomId")
		}

		userID := c.Get("userID").(string)
		err := hub.LeaveRoom(c.Request().Context(), userID, roomID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrForbidden):
				return echo.NewHTTPError(http.StatusForbidden, "error forbidden")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusOK, nil)
	}
}
