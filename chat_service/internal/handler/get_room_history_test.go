package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"chat_service/internal/domain"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GetRoomHistory(t *testing.T) {
	aliceID := "aliceid"
	roomID := "roomid"
	timeNow := time.Now().UTC().Truncate(time.Second)
	dbError := errors.New("db down")

	tests := []struct {
		name       string
		setup      func(h *hubMock, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success last messages",
			setup: func(h *hubMock, c *echo.Context) {
				h.On("GetRoomHistory", mock.Anything, aliceID, roomID, time.Time{}).Return(([]*domain.Message)(nil), nil)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success last messages before",
			setup: func(h *hubMock, c *echo.Context) {
				h.On("GetRoomHistory", mock.Anything, aliceID, roomID, timeNow).Return(([]*domain.Message)(nil), nil)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				c.Request().URL.RawQuery = "before=" + url.QueryEscape(timeNow.Format(time.RFC3339))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid room id",
			setup: func(h *hubMock, c *echo.Context) {
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "invalid before flag",
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				c.Request().URL.RawQuery = fmt.Sprintf("before=%d", timeNow.UTC().UnixNano())
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "room not found",
			setup: func(h *hubMock, c *echo.Context) {
				h.On("GetRoomHistory", mock.Anything, aliceID, roomID, timeNow).Return(([]*domain.Message)(nil), domain.ErrRoomNotFound)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				c.Request().URL.RawQuery = "before=" + url.QueryEscape(timeNow.Format(time.RFC3339))
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			setup: func(h *hubMock, c *echo.Context) {
				h.On("GetRoomHistory", mock.Anything, aliceID, roomID, time.Time{}).Return(([]*domain.Message)(nil), dbError)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &hubMock{}

			_, c, res := newContext(http.MethodGet, "/rooms", "")
			tt.setup(svc, c)
			c.Set("userID", aliceID)

			err := GetRoomHistory(svc)(c)

			if tt.wantErr {
				var echoErr *echo.HTTPError
				require.ErrorAs(t, err, &echoErr)
				assert.Equal(t, tt.wantStatus, echoErr.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, res.Code)
			}
			svc.AssertExpectations(t)
		})
	}
}
