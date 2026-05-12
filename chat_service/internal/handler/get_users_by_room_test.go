package handler

import (
	"chat_service/internal/domain"
	"chat_service/internal/service"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GetUsersByRoom(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(h *service.MockHub, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			setup: func(h *service.MockHub, c *echo.Context) {
				h.EXPECT().GetRoomClients(mock.Anything, roomID).Return([]*domain.User{{Username: "alice"}, {Username: "bob"}, {Username: "clay"}}, nil)

				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid roomId",
			setup: func(h *service.MockHub, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: ""}})
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)

			_, c, res := newContext(http.MethodGet, "/rooms", "")

			tt.setup(svc, c)
			fmt.Println(c.Param("roomID"))
			err := GetUsersByRoom(svc)(c)

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
