package handler

import (
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/service"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func RemoveFriend(svc service.Friendship) echo.HandlerFunc {
	return func(c *echo.Context) error {
		userID := c.Get("userID").(string)
		friendID := c.Param("friendId")

		if _, err := uuid.Parse(friendID); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid friend id")
		}

		err := svc.RemoveFriend(c.Request().Context(), userID, friendID)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrFriendNotFound):
				return echo.NewHTTPError(http.StatusNotFound, "friend not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
			}
		}

		return c.NoContent(http.StatusNoContent)
	}
}
