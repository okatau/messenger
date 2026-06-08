package handler

import (
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/mocks"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	aliceID = uuid.NewString()
	bobID   = uuid.NewString()

	dbError = errors.New("db down")
)

func newContext(method, target, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, bodyReader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func Test_Handler_SendFriendRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(svc *mocks.MockFriendship)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"inviteeId":"` + bobID + `"}`,
			setup: func(svc *mocks.MockFriendship) {
				svc.EXPECT().SendFriendRequest(mock.Anything, aliceID, bobID).Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid json body",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid uuid",
			body:       `{"inviteeId":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid invitee",
			body: `{"inviteeId":"` + bobID + `"}`,
			setup: func(svc *mocks.MockFriendship) {
				svc.EXPECT().SendFriendRequest(mock.Anything, aliceID, bobID).Return(domain.ErrUserNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "request already exists",
			body: `{"inviteeId":"` + bobID + `"}`,
			setup: func(svc *mocks.MockFriendship) {
				svc.EXPECT().SendFriendRequest(mock.Anything, aliceID, bobID).Return(domain.ErrRequestAlreadyExists)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: `{"inviteeId":"` + bobID + `"}`,
			setup: func(svc *mocks.MockFriendship) {
				svc.EXPECT().SendFriendRequest(mock.Anything, aliceID, bobID).Return(dbError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockFriendship(t)
			if tt.setup != nil {
				tt.setup(svc)
			}

			c, rec := newContext(http.MethodPost, "/friends/requests", tt.body)
			c.Set("userID", aliceID)

			err := SendFriendRequest(svc)(c)

			var httpErr *echo.HTTPError
			if errors.As(err, &httpErr) {
				require.Equal(t, tt.wantStatus, httpErr.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}
