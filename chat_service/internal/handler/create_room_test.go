package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	dbError  = errors.New("db down")
	aliceID  = "aliceid"
	bobID    = "bobid"
	roomID   = "roomid"
	roomName = "room-1"
)

func Test_CreateRoom(t *testing.T) {
	room := &domain.Room{
		ID:   roomID,
		Name: roomName,
	}

	tests := []struct {
		name       string
		body       string
		setup      func(h *service.MockHub)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: fmt.Sprintf(`{"name": "%s"}`, roomName),
			setup: func(h *service.MockHub) {
				h.EXPECT().CreateRoom(mock.Anything, roomName, aliceID).Return(room, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalidReqBody",
			body:       "{bad}",
			setup:      func(h *service.MockHub) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "invalid room name",
			body:       fmt.Sprintf(`{"name": "%s"}`, ""),
			setup:      func(h *service.MockHub) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: fmt.Sprintf(`{"name": "%s"}`, roomName),
			setup: func(h *service.MockHub) {
				h.EXPECT().CreateRoom(mock.Anything, roomName, aliceID).Return((*domain.Room)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)

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
		})
	}
}
