package handler

import (
	"errors"
	"net/http"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func GetRoomHistory(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		rawts := c.QueryParam("before")

		roomID := c.Param("roomId")
		rID, err := uuid.Parse(roomID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid room id")
		}

		var history []*domain.Message
		if rawts == "" {
			history, err = hub.GetRoomHistory(c.Request().Context(), rID.String(), userID, time.Time{})
		} else {
			var ts time.Time
			ts, err = time.Parse(time.RFC3339, rawts)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid before flag")
			}
			history, err = hub.GetRoomHistory(c.Request().Context(), rID.String(), userID, ts)
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
