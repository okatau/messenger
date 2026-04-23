package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"friends_service/pkg/token_manager"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var accessTokenTTL = 15 * time.Minute

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

func setupTokenManager(t *testing.T) *token_manager.TokenManager {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	manager, err := token_manager.NewTokenManager(pubPEM, privPEM, accessTokenTTL, slog.Default())
	require.NoError(t, err)
	return manager
}

func Test_Auth(t *testing.T) {
	tokenManager := setupTokenManager(t)
	userID := uuid.NewString()

	validToken, err := tokenManager.GenerateAccessToken(userID)
	require.NoError(t, err)

	next := func(c *echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	}

	tests := []struct {
		name        string
		authHeader  string
		wantStatus  int
		wantErr     bool
		checkUserID bool
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid format - no scheme",
			authHeader: validToken,
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid format - wrong scheme",
			authHeader: "Basic " + validToken,
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token value",
			authHeader: "Bearer invalidtoken",
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:        "valid token",
			authHeader:  "Bearer " + validToken,
			wantErr:     false,
			wantStatus:  http.StatusOK,
			checkUserID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, c, rec := newContext(http.MethodGet, "/test", "")

			if tt.authHeader != "" {
				c.Request().Header.Set("Authorization", tt.authHeader)
			}

			err := Auth(tokenManager)(next)(c)

			if tt.wantErr {
				var echoErr *echo.HTTPError
				require.ErrorAs(t, err, &echoErr)
				assert.Equal(t, tt.wantStatus, echoErr.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
				if tt.checkUserID {
					assert.Equal(t, userID, c.Get("userID"))
				}
			}
		})
	}
}
