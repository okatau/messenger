package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"auth_service/internal/domain"
	"auth_service/internal/handler/mocks"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	dbError = errors.New("db error")

	aliceName = "alice"
	aliceMail = "alice@mail.com"
	alicePW   = "alice"
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
	c := e.NewContext(req, rec)

	return e, c, rec
}

func Test_Login(t *testing.T) {
	getBody := func(email, password string) string {
		body, _ := json.Marshal(
			struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}{
				Email:    email,
				Password: password,
			})

		return string(body)
	}

	tests := []struct {
		name       string
		body       string
		setup      func(s *mocks.MockAuth)
		wantStatus int
		wantError  bool
	}{
		{
			name: "success",
			body: getBody(aliceMail, alicePW),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Login(mock.Anything, aliceMail, alicePW).Return(&domain.AuthSession{AccessToken: "access_token"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid request body",
			body:       "{bad}",
			setup:      func(*mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid email",
			body:       getBody("", alicePW),
			setup:      func(_ *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name:       "invalid password",
			body:       getBody(aliceMail, ""),
			setup:      func(_ *mocks.MockAuth) {},
			wantStatus: http.StatusBadRequest,
			wantError:  true,
		},
		{
			name: "internal server error",
			body: getBody(aliceMail, alicePW),
			setup: func(s *mocks.MockAuth) {
				s.EXPECT().Login(mock.Anything, aliceMail, alicePW).Return((*domain.AuthSession)(nil), dbError)
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockAuth(t)
			tt.setup(svc)

			_, c, rec := newContext(http.MethodPost, "/login", tt.body)

			err := Login(svc)(c)

			if tt.wantError {
				var echoError *echo.HTTPError
				require.ErrorAs(t, err, &echoError)
				assert.Equal(t, tt.wantStatus, echoError.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Contains(t, resp, "accessToken")
			}
		})
	}
}
