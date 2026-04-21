package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"chat_service/internal/domain"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_InviteUser(t *testing.T) {
	aliceID := "aliceid"
	bobID := "bobid"
	roomID := "roomid"
	dbError := errors.New("db down")

	getBody := func(userID string) string {
		body, _ := json.Marshal(
			struct {
				UserID string `json:"userId"`
			}{
				UserID: userID,
			},
		)
		return string(body)
	}

	tests := []struct {
		name       string
		body       string
		setup      func(h *hubMock, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: getBody(bobID),
			setup: func(h *hubMock, c *echo.Context) {
				h.On("InviteUser", mock.Anything, aliceID, bobID, roomID).Return(nil)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "invalid req body",
			body: `{"bad"}`,
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "invalid room id",
			body: getBody(bobID),
			setup: func(h *hubMock, c *echo.Context) {
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "invalid userId",
			body: getBody(""),
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "forbidden",
			body: getBody(bobID),
			setup: func(h *hubMock, c *echo.Context) {
				h.On("InviteUser", mock.Anything, aliceID, bobID, roomID).Return(domain.ErrUserForbidden)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: getBody(bobID),
			setup: func(h *hubMock, c *echo.Context) {
				h.On("InviteUser", mock.Anything, aliceID, bobID, roomID).Return(dbError)
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &hubMock{}
			_, c, rec := newContext(http.MethodPost, "/rooms/", tt.body)
			c.Set("userID", aliceID)

			tt.setup(svc, c)

			err := InviteUser(svc)(c)

			if tt.wantErr {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
			svc.AssertExpectations(t)
		})
	}
}
