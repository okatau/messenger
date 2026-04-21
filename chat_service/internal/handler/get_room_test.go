package handler

import (
	"errors"
	"net/http"
	"testing"

	"chat_service/internal/domain"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GetRoom(t *testing.T) {
	aliceID := "aliceID"

	t.Run("success", func(t *testing.T) {
		_, c, res := newContext(http.MethodGet, "/rooms", "")
		c.Set("userID", aliceID)
		svc := &hubMock{}
		svc.On("GetRoomsByUser", mock.Anything, aliceID).Return(([]*domain.Room)(nil), nil)
		err := GetRoom(svc)(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.Code)
	})

	t.Run("internal server error", func(t *testing.T) {
		_, c, _ := newContext(http.MethodGet, "/rooms", "")
		c.Set("userID", aliceID)
		dbError := errors.New("db down")
		svc := &hubMock{}
		svc.On("GetRoomsByUser", mock.Anything, aliceID).Return(([]*domain.Room)(nil), dbError)
		err := GetRoom(svc)(c)
		var echoError *echo.HTTPError
		require.ErrorAs(t, err, &echoError)
		assert.Equal(t, http.StatusInternalServerError, echoError.Code)
	})
}
