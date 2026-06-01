package handler

import (
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

func Test_CreateDM(t *testing.T) {
	room := &domain.Room{
		ID:   roomID,
		Name: &roomName,
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
			body: fmt.Sprintf(`{"inviteeId": "%s"}`, bobID),
			setup: func(h *service.MockHub) {
				h.EXPECT().CreateDM(mock.Anything, aliceID, bobID).Return(room, nil)
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
			name:       "invalid invitee id name",
			body:       fmt.Sprintf(`{"inviteeId": "%s"}`, ""),
			setup:      func(h *service.MockHub) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "user id equals invitee id",
			body:       fmt.Sprintf(`{"inviteeId": "%s"}`, aliceID),
			setup:      func(h *service.MockHub) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "internal server error",
			body: fmt.Sprintf(`{"inviteeId": "%s"}`, bobID),
			setup: func(h *service.MockHub) {
				h.EXPECT().CreateDM(mock.Anything, aliceID, bobID).Return((*domain.Room)(nil), errDB)
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMockHub(t)

			tt.setup(svc)

			_, c, res := newContext(http.MethodPost, "/rooms/dm/", tt.body)
			c.Set("userID", aliceID)
			err := CreateDM(svc)(c)

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
