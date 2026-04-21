package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"chat_service/internal/domain"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_CreateRoom(t *testing.T) {
	aliceID := "aliceid"
	roomName := "room-1"
	room := &domain.Room{
		ID:   "room1",
		Name: roomName,
	}
	dbError := errors.New("db down")

	tests := []struct {
		name       string
		body       string
		setup      func(h *hubMock)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: fmt.Sprintf(`{"name": "%s"}`, roomName),
			setup: func(h *hubMock) {
				h.On("CreateRoom", mock.Anything, roomName, aliceID).Return(room, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalidReqBody",
			body:       "{bad}",
			setup:      func(h *hubMock) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid room name",
			body:       fmt.Sprintf(`{"name": "%s"}`, ""),
			setup:      func(h *hubMock) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: fmt.Sprintf(`{"name": "%s"}`, roomName),
			setup: func(h *hubMock) {
				h.On("CreateRoom", mock.Anything, roomName, aliceID).Return((*domain.Room)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &hubMock{}

			tt.setup(svc)

			_, c, res := newContext(http.MethodPost, "/rooms/", tt.body)
			c.Set("userID", aliceID)
			err := CreateRoom(svc)(c)

			if tt.wantErr {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, res.Code, tt.wantStatus)
			}
			svc.AssertExpectations(t)
		})
	}
}
