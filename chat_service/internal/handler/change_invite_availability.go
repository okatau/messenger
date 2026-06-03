package handler

import (
	"net/http"

	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
)

func ChangeInviteAvailability(hub service.Hub) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in chat_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)

		var req struct {
			Availability bool `json:"availability"`
		}

		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
		}

		err := hub.ChangeInviteAvailability(c.Request().Context(), userID, req.Availability)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
		}

		return c.NoContent(http.StatusOK)
	}
}
