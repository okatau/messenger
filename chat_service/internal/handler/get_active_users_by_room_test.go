package handler

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GetRoomActiveUsers(t *testing.T) {
	roomID := "roomid"

	tests := []struct {
		name       string
		setup      func(h *hubMock, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			setup: func(h *hubMock, c *echo.Context) {
				h.On("GetRoomClients", mock.Anything, roomID).Return([]string{"alice", "bob", "clay"})
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: "room-1"}})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid roomId",
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: ""}})
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &hubMock{}

			_, c, res := newContext(http.MethodGet, "/rooms", "")

			tt.setup(svc, c)
			err := GetActiveUsersByRoom(svc)(c)

			if tt.wantErr {
				var echoErr *echo.HTTPError
				require.ErrorAs(t, err, &echoErr)
				assert.Equal(t, tt.wantStatus, echoErr.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, res.Code)
			}
		})
	}
}
