package handler

import (
	"chat_service/internal/domain"
	"chat_service/internal/service"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

func GetRoomHistory(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)
		rawts := c.QueryParam("before")
		roomID := c.Param("roomId")
		if roomID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
		}

		var history []*domain.Message
		var err error
		if rawts == "" {
			history, err = hub.GetRoomHistory(c.Request().Context(), userID, roomID, time.Time{})
		} else {
			var ts time.Time
			ts, err = time.Parse(time.RFC3339, rawts)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "before flag")
			}
			history, err = hub.GetRoomHistory(c.Request().Context(), userID, roomID, ts)
		}

		if err != nil {
			switch {
			case errors.Is(err, domain.ErrRoomNotFound):
				return echo.NewHTTPError(http.StatusBadRequest, "room not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.JSON(http.StatusOK, history)
	}
}
