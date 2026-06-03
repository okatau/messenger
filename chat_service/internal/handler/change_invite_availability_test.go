package handler

import (
	"net/http"
	"testing"

	"chat_service/internal/service"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_ChangeInviteAvailability(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(h *service.MockHub)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			body: `{"availability": true}`,
			setup: func(h *service.MockHub) {
				h.EXPECT().ChangeInviteAvailability(mock.Anything, aliceID, true).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "internal server error",
			body: `{"availability": true}`,
			setup: func(h *service.MockHub) {
				h.EXPECT().ChangeInviteAvailability(mock.Anything, aliceID, true).Return(errDB)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)

			tt.setup(svc)

			_, c, res := newContext(http.MethodPost, "/invite-avil", tt.body)
			c.Set("userID", aliceID)
			err := ChangeInviteAvailability(svc)(c)

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
