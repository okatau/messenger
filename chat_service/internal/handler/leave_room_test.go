package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type hubMock struct{ mock.Mock }

func (h *hubMock) InviteUser(ctx context.Context, inviterID, inviteeID, roomID string) error {
	args := h.Called(ctx, inviterID, inviteeID, roomID)
	return args.Error(0)
}

func (h *hubMock) LeaveRoom(ctx context.Context, userID, roomID string) error {
	args := h.Called(ctx, userID, roomID)
	return args.Error(0)
}

func (h *hubMock) CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error) {
	args := h.Called(ctx, roomName, userID)
	return args.Get(0).(*domain.Room), args.Error(1)
}

func (h *hubMock) GetRoomsByUser(ctx context.Context, userID string) ([]*domain.Room, error) {
	args := h.Called(ctx, userID)
	return args.Get(0).([]*domain.Room), args.Error(1)
}

func (h *hubMock) GetRoomHistory(ctx context.Context, userID, roomID string, before time.Time) ([]*domain.Message, error) {
	args := h.Called(ctx, userID, roomID, before)
	return args.Get(0).([]*domain.Message), args.Error(1)
}

func (h *hubMock) Connect(ctx context.Context, userID string, conn *websocket.Conn) error { return nil }
func (h *hubMock) Disconnect(ctx context.Context, userID string) (service.User, error) {
	return nil, nil
}
func (h *hubMock) Shutdown(ctx context.Context)          {}
func (h *hubMock) GetRoomClients(roomID string) []string { return nil }

func newContext(method, target, body string) (*echo.Echo, *echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	return e, e.NewContext(req, rec), rec
}

func Test_LeaveRoom(t *testing.T) {
	aliceID := "aliceid"

	tests := []struct {
		name       string
		setup      func(h *hubMock, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: "room-1"}})
				h.On("LeaveRoom", mock.Anything, aliceID, "room-1").Return(nil)
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
		{
			name: "forbidden",
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: "room-1"}})
				h.On("LeaveRoom", mock.Anything, aliceID, "room-1").Return(domain.ErrUserForbidden)
			},
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name: "internal server error",
			setup: func(h *hubMock, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: "room-1"}})
				h.On("LeaveRoom", mock.Anything, aliceID, "room-1").Return(errors.New("db down"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &hubMock{}
			_, c, rec := newContext(http.MethodDelete, "/rooms/", "")
			c.Set("userID", aliceID)

			tt.setup(svc, c)

			err := LeaveRoom(svc)(c)

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
