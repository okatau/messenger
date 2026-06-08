package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"friends_service/internal/domain"
	"friends_service/internal/service"
)

func RemoveFriend(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		//nolint:errcheck // userID sets in friends_service/internal/middleware/extract_userid.go
		userID := c.Get("userID").(string)
		friendID := c.Param("friendId")

		if _, err := uuid.Parse(friendID); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid friend id")
		}

		err := svc.RemoveFriend(c.Request().Context(), userID, friendID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserNotFound):
				return echo.NewHTTPError(http.StatusNotFound, "friend not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.NoContent(http.StatusNoContent)
	}
}
