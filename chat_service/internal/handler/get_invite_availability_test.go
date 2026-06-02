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

func Test_GetInviteAvailability(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(h *service.MockHub)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "success",
			setup: func(h *service.MockHub) {
				h.EXPECT().GetInviteAvailability(mock.Anything, aliceID).Return(true, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "internal server error",
			setup: func(h *service.MockHub) {
				h.EXPECT().GetInviteAvailability(mock.Anything, aliceID).Return(false, errDB)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)

			tt.setup(svc)

			_, c, res := newContext(http.MethodGet, "/invite-avil", "{}")
			c.Set("userID", aliceID)
			err := GetInviteAvailability(svc)(c)

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
