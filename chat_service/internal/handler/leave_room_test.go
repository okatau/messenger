package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chat_service/internal/domain"
	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	tests := []struct {
		name       string
		setup      func(h *service.MockHub, c *echo.Context)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			setup: func(h *service.MockHub, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				h.EXPECT().LeaveRoom(mock.Anything, aliceID, roomID).Return(nil)
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
		{
			name: "forbidden",
			setup: func(h *service.MockHub, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				h.EXPECT().LeaveRoom(mock.Anything, aliceID, roomID).Return(domain.ErrUserForbidden)
			},
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name: "internal server error",
			setup: func(h *service.MockHub, c *echo.Context) {
				c.SetPathValues(echo.PathValues{{Name: "roomId", Value: roomID}})
				h.EXPECT().LeaveRoom(mock.Anything, aliceID, roomID).Return(dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)
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
		})
	}
}
